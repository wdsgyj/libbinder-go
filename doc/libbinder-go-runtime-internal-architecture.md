# Go 用户态 Binder Runtime 内部架构图

## 1. 目标与边界

本文描述的是：

- 保留现有 Linux/Android kernel Binder driver
- 使用 Go 重写用户态 Binder runtime
- 内部如何分层、如何路由请求、哪些模块必须 thread-bound

不包括：

- 用 Go 重写内核 Binder driver
- AIDL 生成器细节
- Java/NDK 兼容层实现细节

---

## 2. 总体内部架构

```text
应用代码 / 生成代码 / 服务实现
        |
        v
+-----------------------------+
| binder public API           |
|-----------------------------|
| Binder / Parcel / Errors    |
| ServiceManager / Server     |
| context-aware facade        |
+-------------+---------------+
              |
              v
+-----------------------------+
| runtime core                |
|-----------------------------|
| object registry             |
| proxy / stub dispatch       |
| ref tracking                |
| death subscriptions         |
| parcel codec                |
| transaction router          |
+------+----------------------+
       | 
       +--------------------+
       |                    |
       v                    v
+----------------+  +----------------------+
| kernel backend |  | rpc backend          |
|----------------|  |----------------------|
| worker manager |  | session manager      |
| thread-bound   |  | conn pool            |
| ioctl bridge   |  | socket transport     |
| mmap buffers   |  | async goroutine I/O  |
+--------+-------+  +-----------+----------+
         |                      |
         v                      v
   /dev/binder             unix/vsock/tls/...
         |
         v
   kernel Binder driver
```

核心思想：

1. `binder/` 公开层只暴露 Go 风格 API
2. `runtime core` 统一对象模型、路由和错误语义
3. `kernel backend` 与 `rpc backend` 共用上层语义，但底层并发模型不同

---

## 3. 模块拆分图

```text
binder/
├── api facade
│   ├── Binder
│   ├── Handler
│   ├── Parcel
│   ├── ServiceManager
│   └── Subscription
│
├── runtime core
│   ├── node registry
│   ├── handle/address table
│   ├── transact dispatcher
│   ├── reply encoder
│   ├── death watcher
│   ├── ref accounting
│   └── parcel codec
│
├── internal/kernel
│   ├── driver fd manager
│   ├── process state
│   ├── worker pool
│   ├── looper threads
│   ├── client transact threads
│   ├── ioctl bridge
│   └── binder protocol structs
│
├── internal/rpc
│   ├── rpc session
│   ├── rpc server
│   ├── conn pool
│   ├── rpc state
│   └── transports
│
└── internal/protocol
    ├── parcel object encoding
    ├── exception/status mapping
    ├── stability tags
    └── transaction flags/codes
```

---

## 4. Kernel Backend 内部架构

这是最关键的一层，因为它必须遵守现有 kernel Binder driver 的线程语义。

```text
goroutine caller
    |
    | request
    v
+-------------------------+
| transact router         |
|-------------------------|
| choose kernel worker    |
+------------+------------+
             |
             v
+-------------------------+
| client binder worker    |  runtime.LockOSThread()
|-------------------------|
| threadState             |
| out buffer              |
| in buffer               |
| waitForResponse loop    |
+------------+------------+
             |
             v
      ioctl(BINDER_WRITE_READ)
             |
             v
       kernel Binder driver
```

服务端方向：

```text
kernel Binder driver
        |
        v
+-------------------------+
| looper worker           |  runtime.LockOSThread()
|-------------------------|
| BC_ENTER_LOOPER         |
| read BR_TRANSACTION     |
| decode Parcel           |
| invoke local handler    |
| encode reply            |
| write BC_REPLY          |
+------------+------------+
             |
             v
       local service stub
             |
             v
       user handler impl
```

### 4.1 Kernel Backend 组件说明

| 组件 | 责任 |
| --- | --- |
| `driverManager` | 打开 `/dev/binder`、初始化 mmap、管理关闭 |
| `processState` | 进程级 handle 表、context manager、线程池配置 |
| `workerManager` | 管理锁定 OS 线程的 Binder worker |
| `looperWorker` | 作为服务端收包线程，加入 Binder 线程池 |
| `clientWorker` | 发起同步事务并等待 reply |
| `threadState` | 当前绑定线程上的 in/out buffer、事务栈、identity 状态 |
| `kernelCodec` | Go struct 与 binder protocol struct 互转 |

### 4.2 为什么 Kernel Backend 要独立

因为这一层有 3 个特征是 RPC 后端没有的：

1. 要与真实 OS 线程绑定
2. 要维护 driver 视角的线程状态
3. 要处理中途插入的 `BR_*` 命令，而不只是“发请求等响应”

---

## 5. RPC Backend 内部架构

RPC 后端可以明显更 Go-native。

```text
goroutine caller
    |
    v
+-------------------------+
| rpc session manager     |
|-------------------------|
| choose conn             |
| multiplex requests      |
+------------+------------+
             |
             v
+-------------------------+
| rpc conn goroutines     |
|-------------------------|
| read loop               |
| write loop              |
| reply correlation       |
| binder address map      |
+------------+------------+
             |
             v
       unix/vsock/tls transport
```

特点：

- 不要求模拟 C++ TLS 风格
- 更适合 `goroutine + channel + mutex + context`
- 仍然共享上层 `Binder`、`Parcel`、`ServiceManager` 语义

---

## 6. 对象模型图

对外不暴露 `BBinder/BpBinder`，但内部仍然需要“本地节点 / 远端代理”这组语义。

```text
public Binder interface
        |
        +--------------------+
        |                    |
        v                    v
 localBinder            remoteBinder
        |                    |
        |                    +-- kernelRemote
        |                    |     - handle
        |                    |     - worker affinity
        |                    |
        |                    +-- rpcRemote
        |                          - session
        |                          - address
        |
        +-- descriptor
        +-- transact dispatch
        +-- death subscriptions
```

本地对象建议结构：

```text
localBinder
├── descriptor
├── handler
├── feature flags
├── death observers
└── export metadata
```

远端对象建议结构：

```text
remoteBinder
├── backend kind
├── kernel handle or rpc address
├── descriptor cache
├── ref state
└── subscription set
```

---

## 7. Parcel 内部架构

```text
Parcel
├── raw buffer []byte
├── read/write position
├── object table
│   ├── binder refs
│   ├── fd refs
│   └── offsets
├── mode
│   ├── kernel
│   └── rpc
└── flags
    ├── allowFDs
    ├── sensitive
    └── serviceFuzzing/debug
```

建议原则：

1. 公开层只暴露值语义读写接口
2. kernel / rpc 差异放在 internal codec
3. binder object / fd object 永远经过 object table，不直接让业务拼裸布局

---

## 8. 同步事务时序图

### 8.1 客户端同步调用

```text
caller goroutine
    |
    | Transact(ctx, code, parcel)
    v
transaction router
    |
    | dispatch
    v
client binder worker (locked thread)
    |
    | encode BC_TRANSACTION
    | ioctl(BINDER_WRITE_READ)
    | waitForResponse
    | maybe execute BR_* side work
    | decode BR_REPLY
    v
return reply/error
    |
    v
caller goroutine resumes
```

### 8.2 服务端收包处理

```text
looper worker (locked thread)
    |
    | read BR_TRANSACTION
    v
decode incoming parcel
    |
    v
build request context
    |
    v
invoke local handler
    |
    +--> success -> encode reply
    |
    +--> error   -> map to transport error / remote exception
    |
    v
write BC_REPLY
```

---

## 9. ServiceManager 内部架构

```text
public ServiceManager API
        |
        v
+-------------------------+
| sm facade               |
|-------------------------|
| CheckService            |
| WaitService             |
| AddService              |
+------------+------------+
             |
             v
+-------------------------+
| service registry layer  |
|-------------------------|
| local service cache     |
| remote lookup           |
| notification mgmt       |
+------------+------------+
             |
             v
 default servicemanager binder
```

建议：

- ServiceManager 只是普通 Binder 服务的一个特化 facade
- 不要在架构上单独特殊化到不可替换
- 这样以后更容易做 mock、host 适配和测试

---

## 10. Death Notification / Subscription 架构

```text
remoteBinder
    |
    +-- subscription manager
          |
          +-- sub #1 -> done chan
          +-- sub #2 -> callback
          +-- sub #3 -> context cancellation bridge
          |
          v
   backend death source
      ├── kernel: BR_DEAD_BINDER
      └── rpc: session/conn obituary
```

建议：

1. 对外统一成 `Subscription`
2. 对内由 backend 适配不同死亡来源
3. 不要求用户保存 callback identity 去解绑

---

## 11. 建议的 package 视图

```text
doc/
    libbinder-go-runtime-internal-architecture.md

binder/
    binder.go
    handler.go
    parcel.go
    errors.go
    flags.go
    service_manager.go
    server.go
    subscription.go

internal/runtime/
    runtime.go
    registry.go
    router.go
    refs.go
    subscriptions.go

internal/kernel/
    driver.go
    process_state.go
    worker_manager.go
    looper_worker.go
    client_worker.go
    thread_state.go
    codec.go

internal/rpc/
    session.go
    server.go
    state.go
    transport.go

internal/protocol/
    parcel_kernel.go
    parcel_rpc.go
    status.go
    stability.go
    transaction.go
```

---

## 12. MVP 架构切片

第一阶段建议只做下面这些块：

```text
MVP
├── public Binder API
├── public Parcel API
├── runtime registry/router
├── kernel driver manager
├── kernel client worker
├── kernel looper worker
├── ServiceManager facade
└── basic death subscription
```

暂时不做：

```text
Later
├── RPC backend
├── lazy service
├── stability enforcement
├── recording / replay
├── advanced caching
└── Trusty / TLS transport
```

---

## 13. 一句话架构总结

可以把整套 Go 用户态 Binder runtime 概括成：

```text
Go 风格公开 API
    + 统一 runtime core
    + thread-bound kernel backend
    + goroutine-native rpc backend
    + 可替换的 ServiceManager facade
```

其中最重要的非对称点是：

- `kernel backend` 必须尊重真实 OS 线程语义
- `rpc backend` 可以更彻底地使用 Go 并发模型

如果这个边界划清楚，整个实现会稳定很多。
