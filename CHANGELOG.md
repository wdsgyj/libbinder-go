# Changelog

本文件记录当前仓库的重要阶段性产出。

## 0.0.1 - 2026-03-25

首个可运行的 MVP 版本，范围限定为“基于现有 Linux/Android kernel Binder driver 的 Go 用户态实现”，不包含使用 Go 重写内核 Binder。

### Added

- 同步 AOSP `frameworks/native/libs/binder` 源码到 `aosp-src/frameworks/native/libs/binder`，保持相对目录结构，作为分析与实现参考基线。
- 建立 `binder/`、`internal/runtime/`、`internal/kernel/`、`internal/protocol/` 的项目分层与基础骨架。
- 提供公开 Binder API 基础类型：
  - `Binder`
  - `Handler`
  - `Parcel`
  - `ServiceManager`
  - `Subscription`
  - `Flags`
- 提供基础错误模型与远端异常映射，包括：
  - `ErrDeadObject`
  - `ErrFailedTxn`
  - `ErrBadParcelable`
  - `ErrPermissionDenied`
  - `ErrNoService`
  - `RemoteException`
- 实现 `Parcel` MVP 能力：
  - 标量类型读写
  - `string` / `[]byte` 编解码
  - interface token 编码
  - Binder handle 对象解码
  - 本地 Binder object 的 kernel wire format 编码
  - reply/status 基础处理
- 实现 kernel Binder backend 启动链路：
  - 打开 `/dev/binder`
  - 基础 `ioctl` 与 `mmap` 生命周期
  - protocol version / max threads / write-read 桥接
  - `ProcessState`
  - `WorkerManager`
  - 绑定 OS 线程的 `ClientWorker` 与 `LooperWorker`
- 实现客户端事务主链路：
  - 同步 `Transact`
  - `oneway` 事务
  - reply `Parcel` 解码
  - Binder reply object acquire 顺序修正
  - `BR_INCREFS` / `BR_ACQUIRE` / `BR_RELEASE` / `BR_DECREFS` 等基础驱动响应处理
- 实现本地服务端能力：
  - 本地 Binder 节点注册
  - 本地事务分发
  - `INTERFACE_TRANSACTION` / `PING_TRANSACTION` 自动处理
  - looper worker 接收 `BR_TRANSACTION` 并回写 `BC_REPLY`
- 实现 `ServiceManager` MVP：
  - `CheckService`
  - `WaitService`
  - `AddService`
  - 远端 Binder facade 与 `Conn` 公开入口
- 实现 death notification 的最小可用版本：
  - `remoteBinder.WatchDeath(ctx)`
  - handle 级别 death watch 复用
  - `Subscription` fan-out
  - `BR_DEAD_BINDER` / `BR_CLEAR_DEATH_NOTIFICATION_DONE` 处理
  - `BC_DEAD_BINDER_DONE` 回写
- 建立 Android aarch64 模拟器测试脚本与运行说明，支持在模拟器中交叉编译、推送并执行 Go 测试二进制。

### Documentation

- 新增 `doc/libbinder-source-analysis.md`
  - 按逻辑模块梳理 AOSP `libbinder` 源码结构与主干模块。
- 新增 `doc/libbinder-go-rewrite-recommendations.md`
  - 总结使用 Go 重写用户态 `libbinder` 的建议。
  - 重点覆盖线程模型、goroutine 与 TLS 语义差异、内存管理取舍、Go 风格 API 设计。
  - 明确 kernel Binder 仍保留为 Linux 内核层实现，不纳入 Go 重写范围。
- 新增 `doc/libbinder-go-runtime-internal-architecture.md`
  - 给出内部架构图、模块拆分图、时序图与 subscription/death notification 架构。
- 新增 `doc/libbinder-go-mvp-spec.md`
  - 冻结 MVP 范围、能力边界与非目标。
- 新增 `doc/libbinder-go-implementation-roadmap.md`
  - 分阶段给出实现计划、步骤、输出物与完成标准。
- 新增 `doc/android-emulator-testing.md`
  - 记录 Android aarch64 模拟器测试环境、脚本、覆盖范围与当前边界。

### Testing

- 增加 `Parcel` 单元测试，覆盖基础类型、字符串、字节数组、Binder handle 与本地 Binder object wire data。
- 增加 kernel backend 单元测试，覆盖 driver 打开关闭、基础 write/read、context manager ping、本地事务分发。
- 增加 death registry 单元测试，覆盖：
  - handle 级别 watch 复用
  - 死亡通知广播
  - context 取消
  - 请求失败回滚
- 增加 Android 集成测试，覆盖：
  - context manager descriptor
  - `CheckService("activity")`
  - 不存在服务查询
  - `WaitService("activity")`
  - `AddService -> WaitService -> Descriptor -> Transact` 闭环
- 提供 `scripts/android-emulator-test.sh` 与 `scripts/lib/android-emulator-common.sh`，用于 Android aarch64 模拟器自动化验证。

### Verification

- 宿主机测试：
  - `go test ./...`
- Android aarch64 模拟器测试：
  - `ANDROID_AVD_NAME=Medium_Phone ANDROID_SKIP_SDK_INSTALL=1 ANDROID_HEADLESS=1 ANDROID_WIPE_DATA=0 ./scripts/android-emulator-test.sh ./... -- -test.v`

### Current Boundaries

- 仍依赖现有 kernel Binder driver，不实现内核层重写。
- `AddService` 在 stock Android emulator 上可能受 SELinux / service policy 限制，相关用例已按已知策略拒绝场景跳过。
- death notification 目前为最小可用实现，显式 `ClearDeathNotification` 完整退订流与端到端死亡测试仍待补齐。
- 远端对象显式 `Release/Close`、完整引用清理、服务反注册与更完整生命周期管理仍未完成。
- Binder RPC backend、lazy service、stability、record/replay、缓存策略等增强能力尚未纳入本版本。
