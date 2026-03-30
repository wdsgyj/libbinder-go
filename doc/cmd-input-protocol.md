# cmd input Protocol Analysis

本文梳理的是 Android `cmd input` 这一条链路，不是 `IInputManager` 的原始 AIDL 调用。

目标问题有两个：

1. `cmd input` 的请求包到底长什么样。
2. `cmd input` 到底依不依赖 `IShellCallback` / `IResultReceiver` 这条回调链路。

## 源码依据

协议和执行路径主要由这 5 处源码决定：

1. `frameworks/native/cmds/cmd/cmd.cpp`
   - AOSP `cmd` 可执行程序。
   - 会调用 `IBinder::shellCommand(...)` 发起 shell command 事务。
2. `frameworks/native/libs/binder/Binder.cpp`
   - 定义 `IBinder::shellCommand(...)` 的 parcel 编码格式。
   - 定义服务端收到 `SHELL_COMMAND_TRANSACTION` 后的默认解码入口。
3. `frameworks/base/services/core/java/com/android/server/input/InputManagerService.java`
   - `input` service 的 Binder 入口。
   - `onShellCommand(...)` 直接转给 `InputShellCommand.exec(...)`。
4. `frameworks/base/core/java/android/os/ShellCommand.java`
   - 通用 shell command 框架。
   - 保存 `ShellCallback` / `ResultReceiver`，并在命令结束后发送 result code。
5. `frameworks/base/services/core/java/com/android/server/input/InputShellCommand.java`
   - `input` 的实际命令解析和执行实现。
   - 负责 `text` / `keyevent` / `tap` / `swipe` 等命令。

补充一处默认命令逻辑来源：

- `com.android.modules.utils.BasicShellCommandHandler`
  - `cmd == null`、`help`、`-h` 时走 `onHelp()`
  - 未知命令会输出 `Unknown command: ...`
  - 返回值是 `-1`

## 请求结构

`IBinder::shellCommand(...)` 的 request parcel 顺序是固定的：

1. `in` file descriptor
2. `out` file descriptor
3. `err` file descriptor
4. `argc`
5. `argv[0..n-1]`
6. `IShellCallback` binder，可为 `null`
7. `IResultReceiver` binder，可为 `null`

对应上游实现：

- `Binder.cpp`
  - `send.writeFileDescriptor(in/out/err)`
  - `send.writeInt32(numArgs)`
  - `send.writeString16(args[i])`
  - `send.writeStrongBinder(callback or nullptr)`
  - `send.writeStrongBinder(resultReceiver or nullptr)`

这说明：

- `IShellCallback` / `IResultReceiver` 是协议字段
- 但它们是可选字段，不是协议必填

## 端到端链路

完整链路如下：

```text
shell/user
  -> /system/bin/cmd input ...
  -> cmd.cpp
  -> IBinder::shellCommand(...)
  -> Binder transaction: SHELL_COMMAND_TRANSACTION
  -> InputManagerService.onShellCommand(...)
  -> InputShellCommand.exec(...)
  -> ShellCommand.exec(...)
  -> InputShellCommand.onCommand(...)
  -> 输出写到 out/err FD
  -> 可选：通过 ResultReceiver 回传逻辑结果码
```

拆开看：

### 1. `cmd.cpp` 只是 shell command client

`cmd.cpp` 会：

- 解析 `cmd input ...`
- 找到名为 `input` 的 Binder service
- 创建 `MyShellCallback`
- 创建 `MyResultReceiver`
- 调用 `IBinder::shellCommand(...)`

但它有一个关键注释：

- `TODO: block until a result is returned to MyResultReceiver.`

也就是说，当前 AOSP `cmd` 客户端会把 `resultReceiver` 带上，但并不会等待它。

这个细节直接解释了一个现象：

- `cmd input not-a-command` 虽然逻辑上是错误命令
- 但 shell 进程依然可能退出 `0`
- 因为 client 只要 transact 成功就结束了，没有把 `ResultReceiver` 的逻辑结果码回收成进程退出码

### 2. Binder 框架只负责传输，不决定业务是否使用回调

`Binder.cpp` 的 `IBinder::shellCommand(...)` 只是编码 request。

服务端收到 `SHELL_COMMAND_TRANSACTION` 后，会解出：

- `in/out/err`
- `args`
- `shellCallbackBinder`
- `resultReceiver`

是否真的使用这些对象，要看目标服务的 `onShellCommand(...)` 和后续 shell command 实现。

### 3. `input` service 的 Binder 入口非常薄

`InputManagerService.onShellCommand(...)` 只做一件事：

- `new InputShellCommand().exec(this, in, out, err, args, callback, resultReceiver);`

所以 `cmd input` 的业务行为，核心不在 `InputManagerService`，而在 `InputShellCommand` 和 `ShellCommand`。

### 4. `ShellCommand.exec(...)` 管理的是“通用 shell 回调语义”

`ShellCommand.exec(...)` 做两件关键事：

1. 保存 `callback` 和 `resultReceiver`
2. 执行完 `super.exec(...)` 后，如果 `mResultReceiver != null`，就 `send(result, null)`

同时它提供：

- `openFileForSystem(...)`
  - 内部通过 `getShellCallback().openFile(...)`

这说明：

- `ResultReceiver` 是 shell framework 用来回传逻辑 result code 的通用机制
- `ShellCallback` 是 shell framework 提供给服务端“反向要求 shell 打开文件”的通用机制

### 5. `InputShellCommand` 当前实现不依赖 `ShellCallback`

`InputShellCommand.onCommand(...)` 的逻辑是：

1. 解析可选 `<source>`
2. 解析可选 `-d DISPLAY_ID`
3. 根据命令分发到：
   - `runText`
   - `runKeyEvent`
   - `runTap`
   - `runSwipe`
   - `runDragAndDrop`
   - `runPress`
   - `runRoll`
   - `runScroll`
   - `runMotionEvent`
   - `runKeyCombination`
4. 其他情况走 `handleDefaultCommands(arg)`
5. 最终 `return 0`

它的帮助输出使用的是 `getOutPrintWriter()`。

我对照当前 master 没有看到：

- `getShellCallback()`
- `openFileForSystem(...)`

这意味着对当前 `cmd input` 来说：

- `ShellCallback` 在业务上不是必需通道
- 它是协议上可传，但当前命令实现没有依赖它

## 结果码与进程退出码不是一回事

这是 `cmd input` 最容易混淆的地方。

### 服务端逻辑结果码

服务端的 shell command 会有“逻辑返回值”：

- 成功通常是 `0`
- `help` / 未知命令通常经 `handleDefaultCommands(...)` 返回 `-1`

这个值会被 `ShellCommand.exec(...)` 通过 `ResultReceiver.send(...)` 发回去。

### 客户端进程退出码

客户端是否真的把这个 result code 变成进程退出码，要看 client 是否等待了 `ResultReceiver`。

当前 AOSP `cmd.cpp` 没等，所以：

- transact 成功
- shell process 就先退出了
- result code 即使发回来了，也不会影响当前进程退出码

这和协议“支持 resultReceiver”并不矛盾。

## 对 `libbinder-go/cmd/input` 的结论

当前仓库中的 [cmd/input/run.go](../cmd/input/run.go) 采用的是更小的实现：

- 直接按 `SHELL_COMMAND_TRANSACTION` 组包
- 写入 `in/out/err`
- 写入 `argv`
- callback 写 `nil`
- resultReceiver 写 `nil`
- transact 成功即返回 `0`

这样做成立的原因是：

1. `input` 的业务逻辑当前不依赖 `ShellCallback`
2. AOSP `cmd input` 客户端本身也没有等待 `ResultReceiver`
3. 对 shell 用户场景，最关键的是：
   - `stdout/stderr` 能拿到
   - input event 能执行
   - 不要因为本地 callback IPC 链路卡住

## demo 与测试映射

为了把这条链路验证清楚，仓库里对应补了两层验证：

1. `cmd/input`
   - 覆盖真实实现的所有分支
   - 目标是把当前 Go 版 `cmd/input` 自身覆盖做到 100%
2. `demo/cmdinputproto`
   - 不做真实 input 注入
   - 只模拟协议层和 shell command 层语义
   - 验证：
     - request 编解码
     - `source` / `-d DISPLAY_ID` 解析
     - help 路径
     - known command 路径
     - unknown command 路径
     - `ResultReceiver` 发送
     - `ShellCallback` 对 `input` 路径未被使用

## 参考链接

- `cmd.cpp`
  - https://android.googlesource.com/platform/frameworks/native/+/master/cmds/cmd/cmd.cpp
- `Binder.cpp`
  - https://android.googlesource.com/platform/frameworks/native/+/master/libs/binder/Binder.cpp
- `InputManagerService.java`
  - https://android.googlesource.com/platform/frameworks/base/+/master/services/core/java/com/android/server/input/InputManagerService.java
- `InputShellCommand.java`
  - https://android.googlesource.com/platform/frameworks/base/+/master/services/core/java/com/android/server/input/InputShellCommand.java
- `ShellCommand.java`
  - https://android.googlesource.com/platform/frameworks/base/+/HEAD/core/java/android/os/ShellCommand.java
- `BasicShellCommandHandler.java`
  - https://android.googlesource.com/platform/prebuilts/fullsdk/sources/+/dc3f885ebe8ddc75bd9cf2d567eef4d1ed433a09/android-35/com/android/modules/utils/BasicShellCommandHandler.java
