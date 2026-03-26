# Changelog

本文件记录当前仓库的重要阶段性产出。

## Unreleased

### Added

- 增加 Binder 协议兼容审计文档：
  - `doc/binder-protocol-compatibility-audit.md`
  - 梳理 ABI 常量、动态字段、当前风险点与回归矩阵
- 增加 `cmd/cmd` callback 真机回归脚本：
  - `scripts/android-device-cmd-callback-test.sh`
  - 覆盖 `IResultReceiver` 与 `IShellCallback`
  - 验证 `activity help`、`input keyevent 0`、`activity trace-ipc stop --dump-file ...`
- 增加 Binder 协议总回归脚本：
  - `scripts/android-device-protocol-regression.sh`
  - 串行覆盖 `cmd/cmd`、`cmd/input`、`cmd/service`、`cmd/dumpsys` 的真机协议差分
- 增加 AOSP `cmd input` 的 Go 实现：
  - `cmd/input`
  - 通过 `input` service 的 `SHELL_COMMAND_TRANSACTION` 复刻 shell 用户可用的 `cmd input` 主流程
  - 新增单测覆盖参数转发、空参数调用、缺失服务、事务错误映射
- 增加 `cmd input` 协议分析与 demo：
  - `doc/cmd-input-protocol.md`
  - `demo/cmdinputproto`
  - 覆盖 request parcel 结构、help/known/unknown command、`ResultReceiver` 回传与 `ShellCallback` 未使用路径
- 增加真机 `cmd input` 回归脚本：
  - `scripts/android-device-input-test.sh`
  - 默认使用 `keyevent 0` 走真实 input manager 路径，避免明显打扰设备状态
- 增加 AOSP `frameworks/native/cmds/service` 的 Go 实现：
  - `cmd/service`
  - 覆盖 `list` / `check` / `call`
  - `call` 支持 `i32` / `i64` / `f` / `d` / `s16` / `null` / `fd` / `nfd` / `afd` / `intent`
  - 新增单测覆盖参数编解码、FD 传递、intent 编码、reply dump 与帮助输出
- 增加 AOSP `frameworks/native/cmds/dumpsys` 的 Go 实现：
  - `cmd/dumpsys`
  - 覆盖 `-l` / `-t` / `-T` / `--priority` / `--proto` / `--skip`
  - 覆盖 `--dump` / `--pid` / `--stability` / `--thread` / `--clients`
  - 新增单测覆盖参数解析、辅助 dump 类型、timeout、错误继续执行语义
- 增加 `dumpsys` 依赖的 Binder/runtime 调试能力：
  - `binder.DumpBinder`
  - `binder.GetDebugPID`
  - `binder.DebugHandleProvider`
  - local transaction 对 `DUMP_TRANSACTION` / `DEBUG_PID_TRANSACTION` 的保留处理
  - `internal/binderdebug` 对 Android binder proc / transactions 日志的解析

### Changed

- `binder.Parcel.ReadInterfaceToken()` 现在按 AOSP 实际语义读取 request header：
  - 不再把 `strictMode` 与 `workSourceUid` 当作固定值校验
  - 修复 framework callback 在真机上的误判坏 Parcel 问题
- `cmd/cmd` 的 `ShellCallbackHandler.OpenFile()` 现在正确保留绝对路径：
  - 不再把 `/data/local/tmp/...` 这类路径错误拼到 `workingDir` 下
- `ListenRPCUnix("")` 现在会自动分配当前环境可用的临时监听地址：
  - Android 使用 abstract unix socket
  - 其他平台使用临时目录下的 pathname unix socket，并在 `Close()` 时清理
- `TestRPCUnixTransportHelpers` 现在可在 Android aarch64 模拟器上直接执行，不再因测试目录 unix socket 绑定权限而跳过
- 增加 AOSP `frameworks/native/cmds/cmd` 的 Go 实现：
  - `cmds/cmd` 复刻 `-l`、`-w`、shell command transact、`IShellCallback`、`IResultReceiver`
  - `cmds/cmd/cmd` 增加 standalone 二进制入口
  - `binder` 常量补充 `DUMP_TRANSACTION` / `SHELL_COMMAND_TRANSACTION`
  - 新增 host + Android aarch64 模拟器测试覆盖 `cmd` 的主流程与边界行为
- `cmd/dumpsys` 的 worker 执行语义现在与上游更接近：
  - `--thread` / `--clients` 出错时会输出 `NAME_NOT_FOUND` 等状态文本
  - 单项失败不会中断后续 dump 项
  - `stdout` 提前关闭时，非 timeout 场景会等待 worker 收尾，避免额外的 driver close 噪音
- `internal/binderdebug` 的 proc / transactions 打开顺序改为贴近上游：
  - 直接按 binderfs -> debugfs 顺序尝试打开
  - fallback 路径失败时以最后一次打开错误为准

### Testing

- 宿主机：
  - `go test ./...`
- 新增单测覆盖：
  - `cmd/input` 100% statements
  - `demo/cmdinputproto` 100% statements
  - `cmd/dumpsys`
  - `internal/binderdebug`
  - `binder/dump_test.go`

### Verification

- Android arm64 构建：
  - `GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -o /tmp/libbinder-go-cmd ./cmd/cmd`
  - `GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -o /tmp/libbinder-go-service ./cmd/service`
  - `GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -o /tmp/libbinder-go-input ./cmd/input`
  - `GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -o /tmp/libbinder-go-dumpsys ./cmd/dumpsys`
- 真机验证：
  - `./scripts/android-device-cmd-callback-test.sh`
  - `./scripts/android-device-protocol-regression.sh`
  - `/data/local/tmp/libbinder-go-input`
  - `/data/local/tmp/libbinder-go-input not-a-command`
  - `/data/local/tmp/libbinder-go-input keyevent 0`
  - `-l`
  - `--pid activity`
  - `--thread activity`
  - `--clients activity`
  - `-T 2000 activity activities`

## 0.0.7 - 2026-03-26

本版本是在 `0.0.6` 完成既定重写路线图之后，继续针对 AOSP `libbinder` 关键差距做的增强收口，重点补齐：

- `ServiceManager` 高级治理能力
- stability 强制校验与分区语义
- RPC 传输辅助与生命周期语义

### Added

- 扩展 `binder.ServiceManager` 公开能力：
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
- 增加 `ServiceManager` 治理相关公开类型：
  - `binder.ConnectionInfo`
  - `binder.ServiceDebugInfo`
  - `binder.ServiceMetadata`
  - `binder.ServiceRegistration`
  - `binder.ServiceClientUpdate`
- 增加 stability 强校验公开能力：
  - `binder.ErrBadType`
  - `binder.FlagPrivateVendor`
  - `binder.WithRequiredStability`
  - `binder.RequiredStabilityForTransact`
  - `binder.PrepareTransactFlags`
  - `binder.EnforceTransactStability`
  - `binder.ForceDowngradeToLocalStability`
  - `binder.ForceDowngradeToSystemStability`
  - `binder.ForceDowngradeToVendorStability`
  - `binder.RequiresVINTFDeclaration`
- 增加 RPC 连接辅助能力：
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

### Changed

- kernel `ServiceManager` 现在按 Android `IServiceManager.aidl` 事务面补齐治理能力：
  - service list
  - registration notification
  - declared instance 查询
  - APEX / connection info / debug info 查询
  - client callback / try unregister
- RPC `ServiceManager` 现在补齐对应治理语义：
  - 会话内 metadata registry
  - registration watcher
  - client watcher
  - `TryUnregisterService` 约束
- `WaitService` 从固定轮询改为优先基于 registration notification 等待
- `ServiceManager` cache 现在会结合 death notification 进行失效清理
- kernel local binder / kernel remote binder / RPC local binder / RPC remote binder 现在都会在 user transaction 前执行 stability 校验
- RPC 生命周期语义增强：
  - `rpcRemoteBinder.WatchDeath`
  - export 移除时 obituary frame
  - 连接关闭时 imported binder obituary

### Testing

- 新增 root 包单测覆盖：
  - `ServiceManager` 高级治理查询
  - registration notification
  - client callback 与 `TryUnregisterService`
  - RPC transport helpers (`tcp` / `unix` / `tls`)
  - RPC death notification
  - stability enforcement
- 新增 binder 包单测覆盖：
  - `RequiredStabilityForTransact`
  - `EnforceTransactStability`
  - force downgrade helper

### Verification

- 宿主机：
  - `go test ./...`
- Android aarch64 模拟器：
  - `ANDROID_AVD_NAME=Medium_Phone ANDROID_SKIP_SDK_INSTALL=1 ANDROID_HEADLESS=1 ANDROID_WIPE_DATA=0 ./scripts/android-emulator-test.sh ./... -- -test.v`

### Notes

- `TestRPCUnixTransportHelpers` 现在通过 Android abstract unix socket 地址运行，不再因测试目录 pathname unix socket 权限限制而跳过。
- `TestServiceManagerAddServiceAndTransactOnAndroid` 仍受 stock emulator SELinux 限制而跳过。

## 0.0.6 - 2026-03-26

本版本完成了重写路线图最后剩余的“阶段 11：第二阶段增强能力”，至此：

- Go runtime 路线图阶段 1 到阶段 11 已全部完成
- AIDL 全功能计划阶段 0 到阶段 9 继续保持全部完成
- 当前不再存在既定路线图中的未完成阶段，后续工作只属于增量演进

### Added

- 增加 Binder stability 标签能力：
  - `binder.StabilityLevel`
  - `binder.StabilityProvider`
  - `binder.WithStability`
  - `Parcel` Binder object 的 stability 元数据保留与解析
- 增加 lazy service 能力：
  - `binder.NewLazyHandler`
  - `binder.NewLazyHandlerWithMetadata`
  - `AddLazyService`
  - `AddLazyServiceWithMetadata`
- 增加 record/replay 能力：
  - `binder.NewTransactionRecorder`
  - `binder.NewRecordingBinder`
  - `binder.NewReplayBinder`
- 增加 Binder RPC backend：
  - `DialRPC`
  - `ServeRPC`
  - `RPCConn`
  - 会话内 Binder object / callback 透传
  - `RPCConn.DebugSnapshot`
- 增加调试快照能力：
  - `Conn.DebugSnapshot`
  - kernel/runtime/ref/service cache 快照

### Changed

- `ServiceManager` 现在补齐 client cache / addService cache：
  - `CheckService`
  - `WaitService`
  - `AddService`
  都会复用进程内已知服务对象
- remote binder descriptor 现在带缓存：
  - kernel remote binder
  - RPC remote binder
- local dispatch 的保留事务处理被抽成共享逻辑：
  - `binder.DispatchLocalHandler`
  - kernel backend 与 RPC backend 共用
- kernel `BC_ENTER_LOOPER` 兼容性更稳：
  - 接受部分 emulator kernel 返回 `write_consumed=0` 的行为

### Testing

- 新增 binder 单测覆盖：
  - stability 标签与 metadata 保留
  - lazy handler
  - record/replay
- 新增 root 包单测覆盖：
  - `ServiceManager` cache
  - `Conn.DebugSnapshot`
  - RPC service manager / callback / FD 限制 / debug snapshot
- Android aarch64 模拟器回归新增覆盖：
  - lazy service helper
  - RPC backend 基础链路
  - stability / replay / debug snapshot 对应测试包

### Verification

- 宿主机：
  - `go test ./...`
- Android aarch64 模拟器：
  - `ANDROID_AVD_NAME=Medium_Phone ANDROID_SKIP_SDK_INSTALL=1 ANDROID_HEADLESS=1 ANDROID_WIPE_DATA=0 ./scripts/android-emulator-test.sh ./... -- -test.v`

## 0.0.5 - 2026-03-26

本版本完成了“阶段 10 以及之前仍未完成项”的收尾，至此：

- Go runtime 路线图阶段 1 到阶段 10 已全部完成
- AIDL 全功能计划阶段 0 到阶段 9 已全部完成
- 当前只剩阶段 11 的增强能力，不再存在主链缺口

### Added

- 增加 AIDL 表达式与语义能力补齐：
  - `internal/aidl/expr`
  - 更完整的常量表达式解析与求值
  - `parcelable` 内嵌 `enum` / `union` / `parcelable` 解析与 lowering
  - `parcelable` 内部 `const`
  - structured `parcelable` 字段默认值
- 增加生成代码质量回归：
  - `internal/aidl/codegen/testdata/golden/...`
  - `codegen` golden corpus 测试
  - `cmd/aidlgen` 的 AOSP binder corpus 回归
- 增加 checked-in generated fixture：
  - `internal/aidl/generatedfixture`
  - 用于 Android aarch64 模拟器直接执行已生成代码

### Changed

- parser 现在支持：
  - 文件头 block comment
  - 更完整的 expression capture
  - `parcelable` 内 nested decl / const / field default
- resolve 现在补齐 annotation 语义校验：
  - `@nullable`
  - `@nullable(heap=true)`
  - `@Backing(type=...)`
  - `@FixedSize`
  - `@VintfStability`
- gomodel 现在会：
  - 重写常量表达式为 Go 可直接使用的表达式
  - 计算 enum 隐式值
  - 下沉 parcelable const 与字段默认值
- codegen 现在会：
  - 生成 parcelable const
  - 为带默认值的 parcelable 生成 `New<Type>()`
  - 在反序列化入口应用默认值初始化

### Testing

- 新增 parser 单测覆盖：
  - expression capture
  - `parcelable` 内 nested enum / const / default
- 新增 resolve 单测覆盖：
  - annotation 语义校验
  - const expression / fixed-size 校验
- 新增 gomodel 单测覆盖：
  - const expression rewriting
  - nested enum default lowering
- 新增 codegen 单测覆盖：
  - golden corpus
  - parcelable default constructor 生成
- 新增 CLI 单测覆盖：
  - AOSP binder corpus 的 host 侧批量生成
- 新增 generated fixture 测试：
  - host 与 Android emulator 直接执行 checked-in generated code

### Verification

- 宿主机：
  - `go test ./...`
- Android aarch64 模拟器：
  - `ANDROID_AVD_NAME=Medium_Phone ANDROID_SKIP_SDK_INSTALL=1 ANDROID_HEADLESS=1 ANDROID_WIPE_DATA=0 ./scripts/android-emulator-test.sh ./... -- -test.v`

## 0.0.4 - 2026-03-25

本版本补齐了 AIDL 代码生成主链里原先阻塞“完整生成器”落地的几块核心缺口：Binder object / interface / FD 通路、custom parcelable sidecar、stable AIDL 元数据与兼容回退、以及跨文件 import graph 的最小闭包加载。

### Added

- `gomodel` 增加 custom parcelable / stable interface 元数据建模：
  - custom parcelable sidecar 读取
  - stable interface version/hash sidecar 读取
  - `@VintfStability` lowering
  - dependency file 符号收集
- `codegen` 增加完整 AIDL 生成能力补齐：
  - `FileDescriptor`
  - `ParcelFileDescriptor`
  - parcelable/union 内嵌 `IBinder` / interface / FD
  - custom parcelable codec wrapper 生成
  - stable interface version/hash 常量、proxy cache、stub provider、compat fallback
- `cmd/aidlgen` 增加：
  - `-types` sidecar 支持
  - import graph 递归装载
  - 单根文件触发的多文件生成

### Changed

- opaque `parcelable Foo;` 不再在 codegen 阶段一律拒绝，而是通过 sidecar 映射到外部 Go 类型与 codec 入口。
- generated proxy/stub 现在可以正确处理 parcelable/union 中递归出现的 callback/interface 字段，并把 local handler registrar 继续传到嵌套层。
- stable AIDL 保留事务不再只是 runtime 预留能力，生成代码现在会真正暴露：
  - `InterfaceVersion`
  - `InterfaceHash`
  - `UNKNOWN_TRANSACTION` 回退缓存逻辑
- `aidlgen` 不再局限于单文件孤立生成；当 import 目标在源码树可解析时，会一起进入 lowering / codegen 闭包。

### Testing

- 新增 `gomodel` 单测覆盖：
  - `FileDescriptor` / `ParcelFileDescriptor` lowering
  - custom parcelable sidecar lowering
  - stable interface metadata lowering
  - dependency file 解析
- 新增 `codegen` host e2e 覆盖：
  - parcelable/union 中的 interface / `IBinder` / FD round-trip
  - custom parcelable sidecar round-trip
  - stable interface version/hash 与 `UNKNOWN_TRANSACTION` fallback
- 新增 `cmd/aidlgen` 单测覆盖：
  - `-types` sidecar 输出
  - import graph 自动装载并生成依赖文件

### Verification

- 宿主机：
  - `go test ./...`
- Android aarch64 模拟器：
  - `ANDROID_AVD_NAME=Medium_Phone ANDROID_SKIP_SDK_INSTALL=1 ANDROID_HEADLESS=1 ANDROID_WIPE_DATA=0 ./scripts/android-emulator-test.sh ./... -- -test.v`

### Added

- 增加 `internal/aidl/gomodel`，把 AIDL AST 降为 Go backend 可直接使用的 typed model：
  - AIDL -> Go 类型映射
  - nested type flatten
  - interface descriptor / transaction code 分配
  - `in/out/inout` Go 签名建模
  - oneway 约束的基础诊断
- 增加 `internal/aidl/codegen`，支持从 typed model 生成第一版 Go 代码：
  - structured parcelable
  - enum
  - union
  - interface
  - proxy client
  - stub / handler
  - `Check/Wait/AddService` typed helper
- `binder.Parcel` 增加 `ReadInterfaceToken`，用于 generated stub 正确解包和校验请求头。

### Changed

- `cmd/aidlgen` 不再只输出 AST / summary：
  - 新增 `-format model`
  - 新增 `-format go`
  - 新增 `-out` 输出目录支持
- parser/gomodel 现在会把 field 和返回值上的 `@nullable` 下沉到 Go 类型映射。
- enum 成员命名从简单字符串拼接改为更符合 Go 风格的导出名。

### Testing

- 新增 `internal/aidl/gomodel/lower_test.go`
  - 覆盖 nested type、nullable、`inout`、oneway 诊断
- 新增 `internal/aidl/codegen/go_test.go`
  - 生成代码编译测试
  - generated proxy/stub host round-trip 行为测试
  - Android 环境下对 host-only 测试自动跳过
- `cmd/aidlgen/main_test.go` 增加：
  - `model` 输出测试
  - `go` 输出到 stdout 测试
  - `go` 输出到目录测试

### Verification

- 宿主机：
  - `go test ./...`
- Android aarch64 模拟器：
  - `ANDROID_AVD_NAME=Medium_Phone ANDROID_SKIP_SDK_INSTALL=1 ANDROID_HEADLESS=1 ANDROID_WIPE_DATA=0 ./scripts/android-emulator-test.sh ./... -- -test.v`

## 0.0.3 - 2026-03-25

本版本完成了“AIDL 全功能生成器”的基础要求收敛：阶段 0 已冻结，阶段 1 到阶段 3 已具备最小实现骨架，并且对应测试已在宿主机与 Android aarch64 模拟器上通过。

### Added

- 增加 AIDL 前端基础设施：
  - `internal/aidl/ast`
  - `internal/aidl/parser`
  - `internal/aidl/resolve`
  - `internal/aidl/ir`
- 增加最小 `aidlgen` CLI：
  - `cmd/aidlgen`
  - 当前支持输出 AST JSON 和 summary IR JSON
- 增加 `Parcel` 基础类型补齐：
  - `byte`
  - `char`
  - `float`
  - `double`
- 增加 AIDL 集合 codec helper：
  - `WriteSlice` / `ReadSlice`
  - `WriteFixedSlice` / `ReadFixedSlice`
- 增加 AIDL 规划与规范文档：
  - `doc/libbinder-go-aidl-full-plan.md`
  - `doc/libbinder-go-aidl-support-matrix.md`
  - `doc/libbinder-go-aidl-go-backend-mapping.md`
  - `doc/libbinder-go-aidl-custom-parcelable-adapter.md`

### Changed

- 将 AIDL 生成器目标从“功能全集”口号细化为明确的分阶段计划，并冻结了 phase-0 的三类关键规范：
  - support matrix
  - Go backend mapping
  - custom parcelable 适配路径
- `internal/aidl/parser` 现在可稳定解析下面这些核心构造：
  - `package`
  - `import`
  - `interface`
  - `oneway`
  - `const`
  - structured / non-structured `parcelable`
  - `enum`
  - `union`
  - nested type
  - `T[]`
  - `T[N]`
  - `List<T>`
  - annotation 与命名参数
- `internal/aidl/resolve` 增加最小语义校验，当前覆盖重复顶层声明与重复接口成员。
- `internal/aidl/ir` 增加最小 lowering，当前可将主要声明降为 generator 可消费的摘要级 IR。

### Testing

- 增加 `Parcel` 单元测试覆盖：
  - 新增标量 `byte/char/float/double`
  - 可空动态数组与 fixed-size array helper
- 增加 AIDL 前端与 CLI 单元测试覆盖：
  - parser 对 interface/nested type/annotation/array 的解析
  - resolve 重复声明诊断
  - IR lowering
  - `aidlgen` AST / summary 输出

### Verification

- 宿主机：
  - `go test ./...`
- Android aarch64 模拟器：
  - `ANDROID_AVD_NAME=Medium_Phone ANDROID_SKIP_SDK_INSTALL=1 ANDROID_HEADLESS=1 ANDROID_WIPE_DATA=0 ./scripts/android-emulator-test.sh ./... -- -test.v`

### Current Phase

- AIDL 全功能计划中的阶段 0 已完成。
- 阶段 1 到阶段 3 已具备最小基础实现。
- 当前下一步主阻塞项是阶段 4：
  - Binder object
  - interface / callback
  - `IBinder`
  - `FileDescriptor`
  - `ParcelFileDescriptor`

### Current Boundaries

- 当前 `aidlgen` 还不是正式 Go 代码生成器，只提供 parser/IR 调试输出。
- typed IR、正式 proxy/stub 生成、structured parcelable codegen、enum/union codegen、custom parcelable 接入、stable AIDL 兼容语义仍未实现。
- stock Android emulator 上的 `AddService` 仍可能受 SELinux / service policy 限制，相关集成测试继续按已知受限场景跳过。

## 0.0.2 - 2026-03-25

相对 `0.0.1`，本版本主要完成了路线图中的阶段 9 与阶段 10 主体工作，并补充了最小 demo 以及模块/包名规范化调整。

### Changed

- 完成远端 Binder 生命周期管理增强：
  - `Binder` 接口新增 `Close() error`
  - 新增 `binder.ErrClosed`
  - 远端 Binder 支持显式关闭
  - 使用 finalizer 作为远端句柄释放的兜底，而不是主生命周期路径
- 引入进程级 handle 引用跟踪：
  - binder 引用与 death watch 引用统一计数
  - `AcquireHandle` / `ReleaseHandle` 成对工作
  - 处理中途 acquire 与 release 并发交错时的释放时机
- 完成 death notification 退订流：
  - 最后一个订阅关闭时发起 `ClearDeathNotification`
  - 处理 `BR_CLEAR_DEATH_NOTIFICATION_DONE`
  - 订阅关闭后的 handle/watch pin 会正确释放
- `ServiceManager.CheckService` 返回的远端对象现在会立即进入引用跟踪，而不是仅靠后续事务隐式 acquire。
- 顶层 Go 模块名从 `libbinder-go` 改为 `github.com/wdsgyj/libbinder-go`。
- 顶层 Go 包名从 `libbindergo` 改为 `libbinder`。

### Added

- 增加生命周期与引用跟踪实现：
  - `internal/runtime/refs.go`
  - `internal/runtime/handles.go` 中的 `ReleaseHandle`
  - kernel backend 中的 `ReleaseHandle`
- 增加订阅包装层：
  - `subscription_wrapper.go`
  - 用于将 death subscription 的结束和 handle/watch 引用释放绑定起来
- 增加 demo：
  - `demo/echo/server`
  - `demo/echo/client`
  - `demo/echo/README.md`
  - 演示通过 `ServiceManager` 注册一个 echo 服务并由 client 查找后发起事务

### Testing

- 补充生命周期与 death notification 相关单测：
  - `internal/runtime/refs_test.go`
  - `internal/kernel/death_registry_test.go` 中的 clear/unsubscribe 场景
- 补充 Android 集成测试：
  - `WatchDeath + Close`
  - `WatchDeath + context cancel`
- 补充并发与线程绑定测试：
  - client worker 固定 OS 线程验证
  - backend 并发 ping context manager 验证

### Verification

- 宿主机：
  - `go test ./...`
- Android aarch64 模拟器：
  - 生命周期与 death notification 相关改动已通过
  - `ANDROID_AVD_NAME=Medium_Phone ANDROID_SKIP_SDK_INSTALL=1 ANDROID_HEADLESS=1 ANDROID_WIPE_DATA=0 ./scripts/android-emulator-test.sh ./... -- -test.v`

### Current Boundaries

- stock Android emulator 上仍可能因为 SELinux / service policy 拒绝 `AddService`，相关集成测试仍按已知受限场景跳过。
- 目前具备 death notification 注册、退订与本地状态清理，但“真实远端进程退出后收到死亡通知”的端到端 Android 场景还没有稳定自动化用例。
- 服务反注册、更加完整的远端资源清理策略、Binder RPC backend、lazy service、stability、record/replay、缓存策略等增强能力仍未纳入本版本。

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
