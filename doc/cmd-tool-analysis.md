# AOSP `cmd` 工具逻辑分析

## 1. 上游源码范围

本次对齐的 AOSP 源码目录：

- `frameworks/native/cmds/cmd/cmd.cpp`
- `frameworks/native/cmds/cmd/cmd.h`
- `frameworks/native/cmds/cmd/main.cpp`

对应线上源码：

- <https://cs.android.com/android/platform/superproject/+/master:frameworks/native/cmds/cmd/>

---

## 2. 原版 `cmd` 的职责

`cmd` 不是一个 service，本质上是一个 Binder shell-command 前端。

它负责的事情只有四类：

1. 解析命令行
   - `cmd -l`：列出当前运行中的 service
   - `cmd -w <service> ...`：等待 service 出现后再发起 shell command
   - `cmd <service> ...`：直接对 service 发起 shell command

2. 查找 service
   - 通过 `defaultServiceManager()`
   - `-w` 走 `waitForService`
   - 普通模式走 `checkService`

3. 发起 `SHELL_COMMAND_TRANSACTION`
   - request 中依次写入：
     - `stdin` fd
     - `stdout` fd
     - `stderr` fd
     - 参数数组
     - `IShellCallback`
     - `IResultReceiver`

4. 提供两个 callback binder
   - `IShellCallback`
     - service 需要打开额外文件时回调回来
     - `cmd` 在本地做路径拼接、mode 校验、可选的 SELinux access check
   - `IResultReceiver`
     - service 异步回传 shell command 的退出码
     - `cmd` 阻塞等待最终 result

---

## 3. Go 版实现映射

实现目录：

- `cmds/cmd`
- `cmds/cmd/cmd`

### 3.1 入口

- `cmds/cmd/run.go`
  - `Run(ctx, argv, Options) int`
  - `Main(ctx, argv, stdout, stderr) int`
  - `ProcessExitCode(int) int`

- `cmds/cmd/cmd/main.go`
  - standalone 二进制包装
  - 启动时忽略 `SIGPIPE`

### 3.2 Binder 协议层

- `binder/transactions.go`
  - 增加：
    - `DumpTransaction`
    - `ShellCommandTransaction`

- `cmds/cmd/interfaces.go`
  - native `IShellCallback` 的 proxy + handler
  - native `IResultReceiver` 的 proxy + handler
  - 对齐 native C++ 侧的 interface descriptor 和事务号

### 3.3 shell callback 行为

Go 版 `ShellCallbackHandler` 对齐了 `cmd.cpp` 的核心语义：

- 相对路径基于当前工作目录拼接
- 支持四种 mode：
  - `w`
  - `w+`
  - `r`
  - `r+`
- 非法 mode 返回 `EINVAL`
- `Deactivate()` 之后拒绝继续打开文件，返回 `EPERM`
- 预留 `FileAccessChecker` 钩子，用于在 Go 层注入 SELinux / policy 检查

说明：

- AOSP 原版直接调用 `libselinux` 做 `selinux_check_access`
- 当前仓库是纯 Go 路线，因此这里抽象成可注入的 `FileAccessChecker`
- 这样既能覆盖原版逻辑结构，也不会引入当前项目不需要的 cgo 依赖

### 3.4 result receiver 行为

`ResultReceiverHandler` 负责：

- 接收 `send(int32)`
- 记录最终退出码
- 允许 `Run()` 阻塞等待结果

这对应原版 `MyResultReceiver::waitForResult()`

---

## 4. 测试策略

为了让这部分逻辑在宿主机和 Android 模拟器都能稳定回归，Go 版没有把测试绑死到 `/dev/binder`。

而是引入了可测试的 fake 环境：

- fake `ServiceManager`
- fake shell service binder
- fake local binder registrar
- fake binder registry

这样可以覆盖：

- `-l`
- `-w`
- service 缺失
- shell command transact request 编码
- callback/result receiver round-trip
- shell callback 文件打开主路径与拒绝路径
- transact 错误码到用户可读文本的映射

同时，这些测试也已经在 Android aarch64 模拟器上通过，确保新包没有引入平台回归。
