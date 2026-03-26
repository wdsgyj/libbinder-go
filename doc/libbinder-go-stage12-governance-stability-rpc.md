# libbinder-go 0.0.7 治理、稳定性与 RPC 生命周期增强

## 1. 状态

`0.0.7` 是在 `0.0.6` 完成既定重写路线图之后，针对 AOSP `libbinder` 差距继续补齐的一轮增强。

这一版聚焦三块此前最关键的能力缺口：

- `ServiceManager` 高级治理能力
- stability 强制校验与分区语义
- RPC 传输辅助与生命周期语义

---

## 2. ServiceManager 高级治理

当前 `binder.ServiceManager` 公开能力已经扩展为：

- `ListServices`
- `WatchServiceRegistrations`
- `IsDeclared`
- `DeclaredInstances`
- `UpdatableViaApex`
- `UpdatableNames`
- `ConnectionInfo`
- `WatchClients`
- `TryUnregisterService`
- `DebugInfo`

同时补充了相关公开类型：

- `binder.ConnectionInfo`
- `binder.ServiceDebugInfo`
- `binder.ServiceMetadata`
- `binder.ServiceRegistration`
- `binder.ServiceClientUpdate`

设计点：

- kernel Binder `ServiceManager` 直接按 Android `IServiceManager.aidl` 事务号对接
- RPC `ServiceManager` 复用同一套治理语义，并在会话内维护 metadata / watcher / client state
- `WaitService` 改为基于注册通知等待，而不是固定轮询
- cache 现在会结合 death notification 做失效清理

---

## 3. stability enforcement

`0.0.6` 已有 stability 标签；`0.0.7` 补上了真正的 transact 前校验。

新增公开能力：

- `binder.ErrBadType`
- `binder.FlagPrivateVendor`
- `binder.WithRequiredStability`
- `binder.RequiredStabilityFromContext`
- `binder.RequiredStabilityForTransact`
- `binder.PrepareTransactFlags`
- `binder.EnforceTransactStability`
- `binder.ForceDowngradeToLocalStability`
- `binder.ForceDowngradeToSystemStability`
- `binder.ForceDowngradeToVendorStability`
- `binder.RequiresVINTFDeclaration`

当前行为：

- kernel remote binder / local binder / RPC remote binder / RPC local binder 在 user transaction 前都会做 stability 校验
- `FLAG_PRIVATE_VENDOR` 会切换到 vendor required stability
- 也可以通过 `context.Context` 显式覆盖 required stability
- 不满足时返回 `BAD_TYPE`
- RPC 本地 registry 在注册 `VINTF` service 时会要求显式声明

---

## 4. RPC 生命周期与传输辅助

当前新增的 RPC 公开能力：

- `RPCConfig`
- `DialRPCWithConfig`
- `ServeRPCWithConfig`
- `DialRPCNetwork`
- `DialRPCTCP`
- `DialRPCUnix`
- `DialRPCTLS`
- `ListenRPC`
- `ListenRPCTCP`
- `ListenRPCUnix`
- `ListenRPCTLS`
- `AcceptRPC`

生命周期语义补齐包括：

- `rpcRemoteBinder.WatchDeath`
- 连接关闭时 imported binder obituary
- export 显式移除时 obituary frame
- RPC service cache 与 death invalidation 联动
- `WatchClients` 与 `TryUnregisterService` 的会话内状态约束

当前支持的传输矩阵：

- pre-connected `net.Conn`
- `tcp`
- `unix`
- `tls`

---

## 5. 验证

宿主机：

- `go test ./...`

Android aarch64 模拟器：

- `ANDROID_AVD_NAME=Medium_Phone ANDROID_SKIP_SDK_INSTALL=1 ANDROID_HEADLESS=1 ANDROID_WIPE_DATA=0 ./scripts/android-emulator-test.sh ./... -- -test.v`

验证结果：

- 宿主机单元测试通过
- Android 模拟器测试通过
- `TestRPCUnixTransportHelpers` 在 Android 上因测试目录 unix socket 绑定权限受限而按环境差异跳过
- `TestServiceManagerAddServiceAndTransactOnAndroid` 仍因 stock emulator 的 SELinux 限制跳过
