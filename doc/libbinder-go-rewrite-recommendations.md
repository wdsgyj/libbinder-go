# 基于现有 kernel Binder driver 使用 Golang 重写 libbinder 用户态层的设计建议

## 1. 文档目的

这份文档讨论的不是“如何把 C++ `libbinder` 逐文件翻译成 Go”，也不是“如何用 Go 重写 Linux/Android 内核里的 Binder driver”，而是：

- 在保留现有 Linux/Android kernel Binder driver 的前提下，如何用 Go 重写用户态 `libbinder` 对应层
- 在保持 Binder 核心语义的前提下，如何用更符合 Go 风格的方式重构它
- 哪些 C++ 设计应保留语义但放弃原形
- 哪些地方必须尊重 kernel Binder 的底层约束，不能被 Go 抽象掩盖

一句话结论：

`如果底层继续使用现有 kernel Binder driver，那么 Go 版 libbinder 用户态层应保持 Binder 语义兼容，但实现模型必须 Go-native，而不是 C++-shaped。`

### 1.1 范围边界

本文里的几个词，含义固定如下：

- `kernel Binder`
  指 Linux/Android 内核里的 Binder driver，也就是 `/dev/binder` 背后的那套内核实现。
- `Go 重写 libbinder`
  指保留现有 kernel Binder driver，不改内核，只重写用户态 Binder runtime、Parcel、ServiceManager 封装、stub/proxy、AIDL 友好层等。
- `RPC 后端`
  指 `libbinder` 里不依赖 kernel Binder driver 的那套 socket/RPC 路径。

所以本文的前提是：

`不考虑使用 Golang 重写内核 Binder driver，只讨论在现有内核 Binder driver 之上重写用户态层。`

## 2. 总体原则

建议先明确 4 条原则：

1. 保留“协议语义”，不要保留“类层级形状”。
   例如保留 `transact`、death notification、Parcel 编码规则，但不要公开暴露 `BBinder/BpBinder/IPCThreadState` 这种 C++ 结构。
2. 区分“公开 API”与“底层实现约束”。
   公开 API 可以是 goroutine-friendly 的；但当底层继续使用现有 kernel Binder driver 时，用户态与内核交互边界仍然受 OS thread、`ioctl`、`mmap`、线程局部语义约束。
3. 尽量把复杂性压进 `internal/`。
   Go 用户应该看到的是 `Conn`、`Binder`、`Parcel`、`ServiceManager`、`error`、`context.Context`，而不是一堆 driver 细节。
4. 分阶段实现，不追求首版全量对齐。
   首版先做“基于现有 kernel Binder driver 的用户态主链路”和 ServiceManager，再谈 RPC、lazy service、录制回放等扩展功能。

## 3. 最不建议做的事

如果用 Go 重写，最不建议做下面几件事：

1. 原样移植 `RefBase/sp/wp`。
   Go 已有 GC，这套模式会让 API 同时承担两套生命周期模型，结果通常是更难懂。
2. 原样移植 `ProcessState + IPCThreadState + TLS` 到公开层。
   Go 没有“用户可靠可控的线程局部对象”这一套习惯用法。
3. 原样暴露 `BBinder/BpBinder/Bn/BpInterface` 风格 API。
   这会把 Go API 变成“长得像 C++，但失去 C++ 优势，也没得到 Go 优势”。
4. 把 finalizer 当作正确性依赖。
   Go finalizer 只能做兜底回收，不能承载远端引用释放、驱动注销、FD 关闭等关键时序。

## 4. 线程模型：C++ 线程 vs Go goroutine

这是重写时最需要先想清楚的一点。

### 4.1 C++ `libbinder` 的基本假设

当用户态 `libbinder` 基于现有 kernel Binder driver 工作时，C++ `libbinder` 的经典路径建立在两个线程级假设上：

1. `IPCThreadState` 是“每线程一个”
2. 发起事务、等待回复、处理 driver 命令，都是围绕同一个 OS thread 进行的

这对 C++ 很自然，因为：

- 线程就是调度和身份的基本单元
- thread-local storage 很常见
- 引用计数和线程状态绑定比较直接

### 4.2 Go 的根本差异

Go 的 goroutine 会在多个 OS thread 间迁移。

这意味着：

- 你不能假定一个 goroutine 永远绑定到同一个 OS thread
- 你不能把 Binder 的 thread identity 直接等同于 goroutine identity
- 凡是要求“事务必须在同一内核线程里完成”的路径，都不能直接靠普通 goroutine 随便跑

### 4.3 对基于现有 kernel Binder driver 的用户态后端的建议：显式引入“绑定线程层”

如果你的 Go 实现底层继续直接对接 `/dev/binder`，推荐做法不是“让所有 goroutine 直接碰 driver”，而是引入一层显式的“绑定线程执行器”。

建议模型：

```text
goroutine API 层
    |
    v
thread-bound executor
    |
    v
locked OS thread(s)
    |
    v
/dev/binder + ioctl + mmap
```

具体建议：

1. 对 Binder looper 线程使用 `runtime.LockOSThread()`
   每个服务端 binder 线程池 worker 应该是一个长期存活、锁定 OS thread 的 goroutine。
2. 对同步 `transact` 也使用 thread-bound 执行
   不要让“发请求”和“等 reply”分散到不同 OS thread。
3. 不要在公开 API 里暴露“线程状态对象”
   把它封装成 internal 里的 `looperThread`、`clientThread`、`threadState`。
4. `IPCThreadState` 语义可以保留，但应变成显式 struct，而不是 TLS 风格的用户可见单例。

### 4.4 建议的用户态内核后端执行模型

推荐把“基于现有 kernel Binder driver 的用户态执行路径”拆成两类线程：

#### A. 服务端 looper 线程

职责：

- `BC_ENTER_LOOPER` / `BC_REGISTER_LOOPER`
- 阻塞在 driver 上读命令
- 处理 `BR_TRANSACTION`
- 生成 reply
- 回写 driver

建议：

- 固定 N 个锁线程 worker
- handler 默认就在该 looper 线程里执行
- 只有明确需要时再把业务逻辑 offload 到额外 goroutine

原因很简单：

- Binder reply 天然就是收包线程处理最简单
- 如果任意转 goroutine，会把回复路径、取消语义、上下文恢复搞复杂

#### B. 客户端事务线程

职责：

- 发起同步事务
- 等待 `BR_REPLY`
- 处理事务期间收到的其他驱动命令

建议：

- 用一个小型 thread pool 或按需线程对象执行同步 `transact`
- 对单次事务，在整个“write -> wait -> read reply”期间锁定 OS thread

### 4.5 对 Binder RPC 后端的建议

Binder RPC 后端不像“直接对接现有 kernel Binder driver 的用户态后端”那样强依赖 thread-local driver 状态。

因此：

- RPC 后端可以更 goroutine-native
- 连接池、请求并发、回调等待都可以主要用 goroutine + channel + mutex 实现
- 不必为了与现有 kernel Binder driver 的线程语义对齐而强行做 TLS 风格建模

建议把“基于现有 kernel Binder driver 的用户态后端”和 RPC Binder 的并发模型分开设计：

- kernel backend: thread-bound
- rpc backend: goroutine-native

统一它们的应该是上层接口，而不是底层执行模型。

### 4.6 服务实现的并发约束

C++ 服务作者常常默认“线程池会并发进来，但我知道自己在什么线程上”。

Go 里更好的约定是：

1. 服务实现默认必须 goroutine-safe
2. 若某个服务需要串行语义，应显式声明或包一层串行执行器
3. 不要让“线程上下文”成为业务 handler 的隐式依赖

可以考虑给服务注册提供两种模式：

- 默认并发
- 显式串行

而不是让用户自己从线程模型里猜。

### 4.7 为什么不能用 goroutine-local store 替代 thread-local 语义

这个问题需要说清楚，因为它很容易让人误以为：

- 用户态既然本来就有一层 `libbinder`
- `/dev/binder` 又像一个设备边界
- 那是不是可以让 Go runtime 在用户态“伪造”一套线程身份，把 goroutine id 当成 thread id 发给 kernel Binder

答案是不可以。

#### 4.7.1 根本原因：kernel Binder 不接收“用户指定线程 ID”

Binder driver 不是通过协议包里某个“thread id 字段”识别调用线程，而是直接取发起系统调用的真实内核线程，也就是 `current`。

换句话说：

- Binder driver 认的是“哪个真实 OS 线程在调用 `ioctl/poll/read`”
- 不是“用户态说自己是谁”

这意味着用户态根本没有一个入口可以告诉 kernel：

`请把这次调用当成 goroutine 123 对应的 binder 线程`

driver 不会相信，也没有这个协议字段。

#### 4.7.2 kernel Binder 的线程身份来自真实 `current task`

Binder driver 在处理 `/dev/binder` 的 `ioctl` 时，会根据当前真实线程查找或创建 `binder_thread`。

关键点是：

1. binder 进程对象按真实进程建立
2. binder 线程对象按真实线程建立
3. 线程对象里还会保存真实 `task_struct`

也就是说，driver 维护的是：

`proc <-> binder_proc`
`thread(task_struct/current) <-> binder_thread`

而不是：

`用户自定义 ID <-> binder_thread`

因此：

- goroutine id 不能被发送给内核作为 Binder 线程身份
- context value 也不能被发送给内核作为 Binder 线程身份
- 用户态最多只能决定“哪个真实 OS 线程去执行这次 Binder syscall”

#### 4.7.3 为什么“设备边界”没有把线程语义隔离掉

`/dev/binder` 看起来像一个设备接口，但它不是“只收字节流、不看调用者上下文”的纯协议设备。

Binder driver 会结合：

- 当前 file descriptor 所属的 binder proc
- 发起 syscall 的真实内核线程
- 当前线程的 looper 状态
- 当前线程的 transaction stack

来决定事务如何路由、reply 应该回到哪里、当前线程是否在等待、是否允许嵌套事务等。

所以它不是一个“你随便编个虚拟 thread id 给它，它也照单全收”的黑盒。

#### 4.7.4 即使用户态有 goroutine store，也仍然只能做“路由”，不能做“身份替代”

goroutine-local store 最多能做的是：

1. 记录当前 goroutine 应该使用哪个 Binder worker
2. 在用户态把请求路由到对应 worker
3. 让该 worker 在绑定的 OS 线程上执行真正的 Binder syscall

这时 goroutine-local store 的角色是：

- 调度标签
- 路由辅助
- 业务上下文索引

而不是：

- kernel Binder 线程身份
- `IPCThreadState` 的真正语义承载者

也就是说，它可以帮助你找到“去哪儿执行”，但不能改变“真正是谁在执行”。

#### 4.7.5 为什么这不只是“实现细节”

如果错误地把 goroutine-local state 当成 thread-local state，就会出现语义错位：

1. goroutine 迁移线程后，用户态状态和 kernel 看到的线程不一致
2. reply 可能回到另一个真实线程对应的 Binder 状态机
3. looper 注册和事务等待可能落在不同真实线程上
4. 服务端 calling identity 栈无法和真实收包线程对齐
5. 嵌套事务期间，用户态以为自己在“同一上下文”，但 kernel 看的不是同一线程

所以这不是“优雅性问题”，而是 correctness 问题。

#### 4.7.6 正确做法

对基于现有 kernel Binder driver 的 Go 用户态实现，推荐的方式是：

1. 用少量固定 Binder worker 表示真正的 Binder 线程
2. 每个 worker goroutine 启动后立刻 `runtime.LockOSThread()`
3. `IPCThreadState` 等价状态挂在 worker 上，而不是挂在业务 goroutine 上
4. 业务 goroutine 通过 channel / queue / future 把请求路由给 worker
5. `context.Context` 和 goroutine-local store 只用于：
   - 超时/取消
   - trace/logging
   - 请求路由
   - 上层业务元数据

一句话总结：

`goroutine-local store 可以做调度和路由，但不能替代 Binder 所要求的 thread-bound 语义。对于 kernel Binder，真正的身份始终是发起 syscall 的 OS 线程。`

## 5. 内存管理：哪些可以忽略，哪些不能忽略

### 5.1 可以放弃的 C++ 设计

下面这些 C++ 设计不建议在 Go 中照搬：

1. `sp<>` / `wp<>` / `RefBase`
2. 大量“对象析构时自动发送 decStrong/decWeak”的隐式时序
3. 为了所有权设计出来的多层继承结构
4. 过度细粒度的手工生命周期转移

这些东西在 C++ 里有意义，是因为：

- 没有 GC
- 析构函数是核心释放点
- 线程和对象析构时序更容易绑定

Go 不需要复制这套成本。

### 5.2 不能忽略的底层资源

但有一类东西不能“交给 GC 就算了”：

1. FD
2. `mmap` 区域
3. 驱动注册状态
4. 远端 Binder 强弱引用
5. death/freeze notification 注册
6. RPC session / connection 生命周期

这些都是“外部资源”，必须显式管理。

建议原则：

- Go 对象内存可以靠 GC
- 外部资源必须显式 `Close` / `Release` / `Cancel`

### 5.3 对远端引用计数的建议

Binder 协议本身有“强/弱引用”语义，这一点不能因为 Go 有 GC 就假装不存在。

建议做法：

1. 远端引用计数继续存在，但只保留在 internal 层
2. 对外只暴露更简单的概念：
   - `Binder`
   - `Subscription`
   - `Close()`
   - `Release()`
3. 可以用 `runtime.SetFinalizer` 作为兜底释放远端引用，但不能作为主路径

即：

- 正常逻辑依赖显式释放
- finalizer 只防泄漏，不保正确性

### 5.4 对本地对象生命周期的建议

本地服务对象建议直接使用普通 Go 对象。

例如：

- 一个 service struct 被注册到 binder runtime
- runtime 内部把它放到 handle table
- table 对它持有强引用

直到：

- service 被注销
- process 退出
- runtime 关闭

不需要让用户感知“我现在是 strong ref 还是 weak ref”。

### 5.5 Parcel 的内存设计建议

`Parcel` 建议做成明确的 Go struct，而不是类 C++ 的“内部状态机 + 海量重载接口”。

建议形态：

```go
type Parcel struct {
    buf     []byte
    pos     int
    objects []objectRef
    flags   parcelFlags
}
```

建议：

1. 核心数据区使用 `[]byte`
2. Binder/FD/object table 单独维护
3. 只在 internal 层使用 `unsafe`
4. 尽量避免把“借来的驱动 buffer 指针”暴露到公开 API

公开 API 应该让用户看到“读写值”，而不是“直接操作内核布局”。

## 6. API 设计：如何符合 Go 风格

### 6.1 不建议公开暴露 C++ 类名

这些名字可以保留在 internal 注释里，但不建议进入公开 API：

- `BBinder`
- `BpBinder`
- `BnInterface`
- `BpInterface`
- `IPCThreadState`
- `ProcessState`

Go 公开 API 更建议围绕“能力”和“角色”命名：

- `Binder`
- `Proxy`
- `Stub`
- `Conn`
- `Session`
- `ServiceManager`
- `Parcel`

### 6.2 公开 API 应该围绕接口和组合，而不是继承

Go 不需要 C++ 那套“基类 + override `onTransact`”。

建议风格：

```go
type Binder interface {
    Descriptor(ctx context.Context) (string, error)
    Transact(ctx context.Context, code uint32, data *Parcel, flags Flags) (*Parcel, error)
    LinkToDeath(ctx context.Context, h DeathHandler) (Subscription, error)
}
```

本地服务可以是：

```go
type Handler interface {
    Descriptor() string
    HandleTransact(ctx context.Context, code uint32, data *Parcel) (*Parcel, error)
}
```

或者更 Go 一点：

```go
type TransactFunc func(ctx context.Context, code uint32, data *Parcel) (*Parcel, error)
```

然后由 runtime 把它包装成本地 Binder 节点。

### 6.3 用 `context.Context` 统一等待、取消、超时

这几类 API 都建议接受 `context.Context`：

- `Transact`
- `WaitService`
- `Join`
- `Call`
- `Register`

原因：

1. Go 里超时和取消的标准机制就是 `context`
2. 不必单独设计 `waitForServiceWithTimeout`、`tryWait`、`cancelToken`
3. 便于和上层业务整合

例如：

```go
b, err := sm.WaitService(ctx, "activity")
reply, err := b.Transact(ctx, code, req, binder.FlagNone)
```

### 6.4 错误模型建议用 `error`，而不是暴露 `status_t`

公开 API 不建议返回裸 `status_t` 或一堆负数错误码。

建议：

1. 公开 API 返回 `error`
2. 内部保留 binder status code / exception code
3. 对外通过 typed error 包装

例如：

```go
var (
    ErrDeadObject        = errors.New("binder: dead object")
    ErrBadParcelable     = errors.New("binder: bad parcelable")
    ErrPermissionDenied  = errors.New("binder: permission denied")
)

type RemoteException struct {
    Code    ExceptionCode
    Message string
}
```

推荐区分两类错误：

- 传输错误
- 远端业务异常

不要把它们混成一个 `int32` 给用户自己猜。

### 6.5 选项设计建议使用 option pattern，但不要滥用

对于配置项较多的对象，比如 session、server、service 注册，建议使用 option pattern。

例如：

```go
srv, err := binder.NewServer(
    binder.WithMaxThreads(8),
    binder.WithDriver("/dev/binder"),
)
```

但对于简单 API，不建议滥用 option pattern。

例如：

- `WriteInt32(v int32)` 就应该直接传值
- `Descriptor(ctx)` 就不需要再搞一个 `WithDescriptorFlags(...)`

### 6.6 命名建议遵守 Go 常见习惯

建议：

- 不要 `GetXxx`，直接 `Descriptor()`、`Driver()`、`Remote()`
- 使用 `Close()`、`Release()`、`Cancel()` 表达生命周期结束
- 使用 `WriteInt32`、`ReadInt32`，不要搞模板重载风格接口
- 导出名简洁，避免 `IServiceManager` 这种 Java/C++ 命名习惯进入 Go 公开层

例如更推荐：

- `ServiceManager`
- `WaitService`
- `CheckService`
- `RegisterService`

而不是：

- `IServiceManager`
- `DefaultServiceManagerInstance`
- `GetServiceBlocking`

### 6.7 death recipient 更建议做成订阅对象

C++ 的 `linkToDeath/unlinkToDeath` 是典型 callback 注册模型。

Go 里更建议做成订阅对象或 channel：

```go
sub, err := binder.WatchDeath(ctx)
defer sub.Close()

select {
case <-sub.Done():
case <-ctx.Done():
}
```

或者：

```go
sub, err := binder.LinkToDeath(ctx, func() {
    ...
})
```

但无论哪种方式，都建议返回一个可取消的对象，而不是要求用户手动保存原 callback 引用去解绑。

## 7. Parcel / 接口生成 / 代码组织建议

### 7.1 Parcel API 不要模仿 C++ 的全量重载表

C++ `Parcel` 里有大量重载和历史兼容接口。

Go 中不建议照搬。

建议 Parcel 公开层只保留一组清晰方法：

- 标量读写
- 字符串读写
- `[]T` 读写
- `Binder` / `FD` / `Parcelable` 读写
- 必要的 nullable 读写

### 7.2 对可序列化对象使用接口，而不是继承树

建议：

```go
type Parcelable interface {
    MarshalParcel(*Parcel) error
    UnmarshalParcel(*Parcel) error
}
```

必要时可以把反序列化拆成单独接口，但总原则仍然是接口组合。

### 7.3 公开 API 面向 AIDL 生成器，而不是手写事务用户

Go 版真正的主要用户大概率不是手写 `Transact(code, parcel)` 的人，而是：

- AIDL 生成代码
- service 实现者
- runtime 框架作者

因此建议：

1. 核心库提供足够低层 API
2. 但公开设计优先服务“生成代码好用”
3. 不要为了兼容手写事务，把高层 API 设计成“泛 C++ 风格”

### 7.4 包结构建议

推荐把项目拆成这几层：

```text
binder/
    binder.go
    parcel.go
    errors.go
    service_manager.go
    server.go
    flags.go

internal/kernel/
    driver.go
    thread.go
    looper.go
    refs.go
    parcel_kernel.go

internal/rpc/
    session.go
    server.go
    state.go
    transport.go

internal/protocol/
    stability.go
    exception.go
    transaction.go

aidl/
    // 生成代码或生成时代码
```

关键点：

- `binder/` 给用户
- `internal/kernel` 和 `internal/rpc` 给 runtime
- 不要把 driver 布局和用户 API 混在一起

## 8. 哪些 libbinder 特性建议首版不做

如果目的是“先做出可用的 Go 版 libbinder 用户态层”，建议第一阶段不要追以下能力：

1. Binder RPC 全量支持
2. TLS / Trusty 适配
3. 录制回放
4. addService cache / client cache
5. freeze notification
6. 完整稳定性矩阵
7. 所有历史兼容接口

建议首版聚焦：

1. 基于现有 kernel Binder driver 的基本通信
2. `Parcel` 核心编码
3. 本地 stub / 远端 proxy 基础抽象
4. `ServiceManager` 基础注册和查询
5. death notification
6. oneway 与同步调用

先把主链路做稳定，再决定哪些“现代增强特性”值得跟。

## 9. 一个更 Go 风格的 API 草图

下面只是示意，不是最终签名：

```go
package binder

type Flags uint32

const (
    FlagOneway Flags = 1 << iota
)

type Binder interface {
    Descriptor(ctx context.Context) (string, error)
    Transact(ctx context.Context, code uint32, data *Parcel, flags Flags) (*Parcel, error)
    WatchDeath(ctx context.Context) (Subscription, error)
}

type Subscription interface {
    Done() <-chan struct{}
    Close() error
}

type Handler interface {
    Descriptor() string
    HandleTransact(ctx context.Context, code uint32, data *Parcel) (*Parcel, error)
}

type ServiceManager interface {
    CheckService(ctx context.Context, name string) (Binder, error)
    WaitService(ctx context.Context, name string) (Binder, error)
    AddService(ctx context.Context, name string, h Handler, opts ...AddServiceOption) error
}

type Parcel struct {
    // hidden fields
}
```

这个草图表达的是几个方向：

1. 上层围绕 `Binder` 和 `Handler`，不是 `BpBinder/BBinder`
2. 生命周期和取消基于 `context`
3. 错误统一走 `error`
4. 订阅对象显式可关闭

## 10. 实现路线建议

### 阶段 1：基于现有 kernel Binder driver 的最小可用用户态版

目标：

- 打开并管理现有 Binder driver
- 发起/接收事务
- `Parcel` 核心能力
- `ServiceManager` 基础查询/注册
- 简单 death notification

### 阶段 2：Go 风格公开 API 定型

目标：

- 清理公开命名
- 引入 `context`
- typed error 定型
- handler / stub / proxy 结构稳定

### 阶段 3：AIDL 生成器友好化

目标：

- 让生成代码不必手写接口 token、exception 处理、reply 解码
- 补齐 nullable / list / parcelable 能力

### 阶段 4：高级能力

目标：

- Binder RPC
- lazy service
- 稳定性标签
- 录制回放
- 更细粒度调试工具

## 11. 风险点与踩坑提醒

### 11.1 最大风险：低估“现有 kernel Binder driver + 用户态实现”之间的 OS thread 约束

这是最容易出大问题的地方。

如果让 goroutine 自由迁移，而你的事务依赖“同一内核线程收 reply”，就会出现非常隐蔽的错误：

- 回复收不到
- 状态串线
- looper 行为异常
- 偶发死锁

所以要把“thread-bound 执行”当成“用户态对接现有 kernel Binder driver”时的一等概念。

### 11.2 第二个风险：把 GC 当协议资源管理

GC 解决的是 Go 堆对象内存，不是 Binder 协议资源。

如果用“对象没人引用了，等 finalizer 慢慢回收”来处理：

- 远端强引用释放
- FD 关闭
- death recipient 注销

通常都会留下时序 bug。

### 11.3 第三个风险：公开 API 过度贴近 C++

如果 Go API 长成下面这样：

- `NewBpBinder`
- `AsInterface`
- `OnTransact`
- `ProcessStateSelf`

那么你大概率只是把 C++ API 包了一层 Go 语法糖，而不是做了真正可维护的 Go runtime。

## 12. 最终建议

如果用 Go 重写 `libbinder`，我建议用下面这套判断标准：

1. 协议必须兼容
   Binder 对象语义、Parcel 编码、事务时序、ServiceManager 交互不能乱改。
2. 对接现有 kernel Binder driver 的边界必须尊重线程约束
   这个 thread identity 不是 Go runtime 能帮你抽象掉的。
3. 内存管理要分层
   Go 对象交给 GC，外部资源走显式释放。
4. 公开 API 必须 Go-native
   `context`、`error`、interface、组合、订阅对象，应该成为第一选择。
5. 实现要分阶段
   先把“基于现有 kernel Binder driver 的用户态主链路”做好，再逐步补齐 RPC、lazy service、稳定性等高级能力。

最重要的一句建议是：

`不要把 C++ libbinder 翻译成 Go；要在保留现有 kernel Binder driver 的前提下，把 Binder 用户态语义重新设计成 Go runtime。`
