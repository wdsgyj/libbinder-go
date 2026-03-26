# Binder 协议兼容审计

本文整理当前仓库里最容易出现“Parcel / Binder 协议解析错误”的边界，并给出对应的回归矩阵。

目标不是把所有字段都做成“完全不校验”，而是把真正的 ABI 常量和运行时可变字段分开处理：

- ABI 常量可以严格校验
- 运行时上下文字段应当容忍不同取值，或按长度/结构跳过
- 每个手写协议入口都应该至少有单测和一条真机回归

## 本次问题的根因

`cmd/cmd` 的 `IResultReceiver` / `IShellCallback` 在真机上已经收到了 Binder callback，但在进入本地 handler 之前就被 `ReadInterfaceToken()` 误判为坏 Parcel。

原因是 `binder/parcel.go` 之前把 request header 中的两个运行时字段写死成固定值校验：

- `strictMode == 0x80000000`
- `workSourceUid == -1`

真实设备上来自 framework 的 callback 请求里，`workSourceUid` 可以是 `2000` 等实际调用方 UID。AOSP `Parcel::enforceInterface()` 也不会把这个字段当作固定常量校验，它只会读取并保存到 `IPCThreadState`。

另外，`cmd/cmd` 的 `ShellCallbackHandler.OpenFile()` 之前会把绝对路径也拼到 `workingDir` 下，导致 `/data/local/tmp/...` 这类真实路径一定失败。

## 审计原则

新增或修改协议解析代码时，优先遵守以下规则：

1. 成对阅读 AOSP 的 writer / reader 实现
2. 只把真正的协议常量当成强校验项
3. 对运行时字段做“读取并跳过”或“读取并透传”
4. 不猜测可选头长度，优先按 header size 前移
5. 每个手写协议入口至少补：
   - 单元测试
   - 一条真机回归
6. 真机回归优先做“系统二进制 vs 本项目二进制”的差分验证

## 协议常量与动态字段

### 可以严格校验的常量

- Interface token header 中的 `SYST`
- Binder flat object type:
  - strong binder
  - strong handle
  - file descriptor
- `TFStatus`
- `ExceptionNone`
- `ExceptionServiceSpecific`
- `SHELL_COMMAND_TRANSACTION`
- `DUMP_TRANSACTION`
- `DEBUG_PID_TRANSACTION`

### 不能写死为固定值的字段

- request header 里的 `strictMode`
- request header 里的 `workSourceUid`
- `binder::Status` 里的 reply/appops 可选 header 长度
- framework 在 callback 事务里附加的上下文值
- 某些服务根据平台差异返回的 descriptor / debug 信息内容

## 当前审计矩阵

| 模块 | 入口/协议 | 现状 | 风险 | 当前回归 |
| --- | --- | --- | --- | --- |
| `binder/parcel.go` | `ReadInterfaceToken` | 已按 AOSP 语义放宽，读取 `strictMode/workSourceUid` 但不做固定值校验 | 中 | 单测 + 真机 `cmd` callback |
| `binder/parcel.go` | `WriteInterfaceToken` | 当前仍输出 canonical header：`strictMode=0x80000000`、`workSourceUid=-1` | 中 | 单测 |
| `internal/protocol/status_codec.go` | `android::binder::Status` | 已覆盖 OK / reply header / appops header / service specific / parcelable | 中 | 单测 |
| `cmd/cmd/interfaces.go` | `IShellCallback.openFile` | 已修正绝对路径处理；reply 形状与 native C++ 对齐 | 中 | 单测 + 真机 `trace-ipc stop --dump-file` |
| `cmd/cmd/interfaces.go` | `IResultReceiver.send` | 与 native C++ wire 对齐 | 中 | 单测 + 真机 `activity help` / `input keyevent 0` |
| `cmd/cmd/run.go` | shell command request parcel | callback/result binder 真实工作 | 中 | 单测 + 真机差分 |
| `cmd/input/run.go` | shell command request parcel | 当前故意写入 `nil callback` / `nil result`，依赖 `input` 的同步 shell path | 中 | 单测 + 真机差分 |
| `cmd/service/run.go` | `service call` 参数编码 | 常见标量 / FD / intent 已覆盖 | 中 | 单测 + 真机 smoke |
| `cmd/dumpsys/run.go` | dump request / `--pid` / `-l` | 与系统输出基本一致 | 中 | 单测 + 真机 smoke |
| `binder/parcel.go` | weak binder / weak handle 高层读写 | 内核 object 类型可识别，但高层公开 API 仍不是完整重点路径 | 中 | 单测有限 |

## 仍然存在的“写死”点

下面这些地方是“有意识的固定实现”，不是已知 bug，但需要记账：

### 1. `WriteInterfaceToken()` 还没有上下文化输出

当前 `binder/parcel.go` 仍然固定写：

- `strictMode = 0x80000000`
- `workSourceUid = -1`

这通常能工作，因为大多数客户端请求并不依赖这两个字段的动态传播。但它与 AOSP `IPCThreadState` 的完整语义还不等价。

如果后续要对齐得更深入，建议新增：

- request header context 类型
- `WriteInterfaceTokenWithContext(...)`
- 从调用上下文注入 strict mode / work source 的机制

### 2. `cmd/input` 仍然依赖“当前 AOSP input shell 路径不走 callback”

这不是解析 bug，而是刻意简化。当前真机验证已经证明它在现有设备上成立，但将来同步 AOSP 时需要重新确认 `InputShellCommand` 是否仍然不依赖 `IShellCallback` / `IResultReceiver`。

### 3. `Status` 解析虽然单测充分，但真机语料还不够多

目前 `internal/protocol/status_codec.go` 的主要覆盖来自：

- 单元测试
- 若干命令工具的真实调用

后续仍建议补更多真实 reply 语料，尤其是：

- appops reply header
- parcelable exception
- service specific error

### 4. 弱引用 Binder 对象不是当前重点路径

当前仓库已经能在低层识别 weak binder / weak handle object type，但高层 API 和工具链并没有把这条链路当作主要目标。如果后续引入依赖 weak binder 的真实服务，再补专项回归更合适。

## 已落地回归

### 单元测试

- `binder/parcel_test.go`
  - interface token canonical header
  - framework-style dynamic header
- `cmd/cmd/run_test.go`
  - `IShellCallback`
  - `IResultReceiver`
  - 绝对路径文件打开
- 既有测试：
  - `internal/protocol/status_codec_test.go`
  - `cmd/service/run_test.go`
  - `cmd/dumpsys/run_test.go`
  - `cmd/input/run_test.go`

### 真机脚本

- `scripts/android-device-input-test.sh`
  - `cmd/input` 与系统 `/system/bin/cmd input` 差分
- `scripts/android-device-cmd-callback-test.sh`
  - `activity help`
  - `input keyevent 0`
  - `activity trace-ipc stop --dump-file ...`
- `scripts/android-device-protocol-regression.sh`
  - 串行执行 `cmd` callback 回归
  - 串行执行 `cmd/input` 回归
  - 对比 `service check/list`
  - 对比 `dumpsys -l/--pid`

## 推荐的新增协议开发流程

以后如果要新增一个新的 Binder 接口、callback 或命令工具，建议固定按下面步骤落地：

1. 先找 AOSP 的 writer 和 reader 两端源码
2. 标出：
   - 常量字段
   - 长度字段
   - 可变上下文字段
3. 写 host 单测，先覆盖静态 wire shape
4. 能差分系统二进制的，先做系统差分脚本
5. 真机跑通后再做抽象或封装
6. 把已知“固定假设”记入本文档

## 当前结论

当前仓库里“最危险的写死解析”已经修正，`cmd` callback/runtime 真实设备链路也已经恢复。

后续优先级建议如下：

1. 为 `WriteInterfaceToken()` 增加上下文化输出能力
2. 为 `Status` 增加更多真实 reply 语料回归
3. 继续把新增的手写协议入口纳入真机差分脚本
