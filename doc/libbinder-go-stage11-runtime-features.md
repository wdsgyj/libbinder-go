# libbinder-go 阶段 11 增强能力落地说明

## 1. 状态

`0.0.6` 已完成重写路线图中的阶段 11，当前路线图 11 个阶段全部结束。

这一阶段新增的能力包括：

- stability 标签
- lazy service
- client cache / addService cache
- record/replay
- Binder RPC backend
- 调试快照
- 一项明确的性能优化

---

## 2. stability 标签

当前公开能力：

- `binder.StabilityLevel`
- `binder.StabilityProvider`
- `binder.WithStability`
- `binder.CheckStability`

当前行为：

- `Parcel` 中的 Binder object 会保留并回传 wire stability
- kernel local binder / remote binder / RPC binder 都会携带 stability
- `ServiceManager` 返回 Binder 时不会再丢失该元数据

---

## 3. lazy service

当前公开能力：

- `binder.NewLazyHandler`
- `binder.NewLazyHandlerWithMetadata`
- `AddLazyService`
- `AddLazyServiceWithMetadata`

设计点：

- descriptor 可在未初始化真实 handler 时直接返回
- 首次事务才触发 factory
- 若需要 stable-AIDL version/hash，可通过 metadata 直接声明

---

## 4. cache

当前缓存包括：

- kernel `remoteBinder.Descriptor()` cache
- RPC `rpcRemoteBinder.Descriptor()` cache
- `ServiceManager.CheckService/WaitService/AddService` 的进程内 cache

这样可以减少：

- 重复 `INTERFACE_TRANSACTION`
- 重复 service lookup
- 同进程内 add/check 的无意义往返

---

## 5. record/replay

当前公开能力：

- `binder.NewTransactionRecorder`
- `binder.NewRecordingBinder`
- `binder.NewReplayBinder`

当前记录内容：

- descriptor
- transaction code / flags
- request/reply parcel bytes
- object table
- 远端异常或 transport error

用途：

- 回归测试
- 问题复现
- 脱离真实 Binder 端点的 deterministic replay

---

## 6. Binder RPC backend

当前公开能力：

- `DialRPC`
- `ServeRPC`
- `RPCConn`

当前支持：

- 基于 `net.Conn` 的双向 Binder 风格事务
- 会话内 Binder object 传递
- callback binder 透传
- RPC 侧 `ServiceManager`
- RPC 侧 debug snapshot

当前明确边界：

- RPC backend 不支持 file descriptor 传输
- 只能传递当前 RPC session 所拥有或导入的 Binder object
- 不能把 kernel Binder local object 直接塞进 RPC parcel

这与当前实现目标一致：

- 先完成 Binder object / callback / service management 主链
- FD over RPC 留给后续协议增强，而不是当前路线图缺口

---

## 7. 调试与性能

调试能力：

- `Conn.DebugSnapshot()`
- `RPCConn.DebugSnapshot()`

快照内容包括：

- worker / process / local node / death registry 状态
- ref tracker 状态
- service cache 统计
- RPC exported/imported object 数量
- pending call 数量

性能优化：

- RPC frame 采用 `sync.Pool` 复用
- descriptor / service lookup cache 降低重复事务

---

## 8. 验证结果

宿主机：

- `go test ./...`

Android aarch64 模拟器：

- `ANDROID_AVD_NAME=Medium_Phone ANDROID_SKIP_SDK_INSTALL=1 ANDROID_HEADLESS=1 ANDROID_WIPE_DATA=0 ./scripts/android-emulator-test.sh ./... -- -test.v`

两套验证在 `0.0.6` 均已通过。
