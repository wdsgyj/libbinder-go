# Go 用户态 libbinder MVP 规格

## 1. 目的

这份文档冻结首版 MVP 边界，避免后续设计和实现持续漂移。

MVP 的目标不是覆盖全部 AOSP `libbinder` 能力，而是：

- 在现有 Linux/Android kernel Binder driver 之上
- 做出一套最小可用、可验证、可继续迭代的 Go 用户态 Binder runtime

---

## 2. MVP 边界

### 2.1 明确前提

MVP 只讨论：

- 现有 `/dev/binder` kernel Binder driver
- Go 用户态 runtime
- Go 风格公开 API

MVP 不讨论：

- 用 Go 重写内核 Binder driver
- 首版支持 Binder RPC
- 首版支持所有 AIDL 生成器能力

### 2.2 MVP 成功定义

如果下面这些条件同时满足，就认为 MVP 完成：

1. 能打开并管理 `/dev/binder`
2. 能发起同步 Binder 事务
3. 能发起 oneway Binder 事务
4. 能注册本地 Go 服务
5. 能通过 `ServiceManager` 查找和注册服务
6. 能收到基础 death notification
7. 有自动化测试覆盖主链路

---

## 3. MVP 功能范围

### 3.1 In Scope

- `Parcel` 基础容器
- 基础标量和 `[]byte` 编解码
- Binder 对象传递基础路径
- 同步 `Transact`
- oneway `Transact`
- 本地服务 `Handler` 分发
- `ServiceManager`：
  - `CheckService`
  - `WaitService`
  - `AddService`
- 基础 death notification
- Go 风格错误模型
- `context.Context` 驱动的超时/取消

### 3.2 Out of Scope

- Binder RPC 后端
- lazy service
- stability enforcement
- record/replay
- addService cache / client cache
- Trusty 适配
- TLS / socket transport
- 完整 parcelable 生态
- AIDL 代码生成器
- 高级调试工具

---

## 4. MVP 运行时约束

### 4.1 并发模型

MVP 必须遵守：

- kernel Binder 后端采用 thread-bound worker
- Binder looper worker 使用 `runtime.LockOSThread()`
- 同步事务在绑定 OS 线程的 worker 上完成

MVP 不接受以下做法：

- 用普通 goroutine 直接自由对接 `/dev/binder`
- 用 goroutine-local state 替代 thread-bound 语义
- 把 finalizer 当成主生命周期机制

### 4.2 生命周期约束

MVP 必须明确区分：

- Go 堆对象内存：由 GC 管理
- 外部资源：由显式 `Close/Release` 管理

外部资源包括：

- driver fd
- mmap
- 远端引用
- death subscription

---

## 5. MVP 公开 API 目标

MVP 首版 API 应至少覆盖以下概念：

- `Binder`
- `Handler`
- `Parcel`
- `ServiceManager`
- `Subscription`
- `Flags`

API 目标不是模仿 C++ 命名，而是：

- 用 Go 风格接口
- 使用 `context.Context`
- 用 `error` 暴露失败语义

---

## 6. MVP 测试要求

MVP 至少需要下面几类测试：

1. `Parcel` 单元测试
2. Binder 主链路测试
3. 本地服务分发测试
4. `ServiceManager` 查询/注册测试
5. death notification 测试
6. 并发与线程绑定测试

建议区分：

- 不依赖真实 Binder driver 的快速单元测试
- 依赖真实 Binder driver 的集成测试

---

## 7. MVP 完成后的下一步

MVP 完成后，再进入第二阶段能力：

1. 完善 `Parcel` 类型系统
2. 增加稳定性标签
3. 补齐 lazy service
4. 加入 Binder RPC 后端
5. 为 AIDL 生成器提供更完整支撑

---

## 8. 一句话冻结结论

MVP 的目标不是“做一个功能很多的原型”，而是：

`做一个边界清晰、主链路完整、可自动化验证的 Go 用户态 Binder runtime 最小版本。`
