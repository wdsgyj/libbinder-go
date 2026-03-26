# AOSP `dumpstate` 程序工作流与逻辑分析

## 1. 上游源码范围

本次分析基于 AOSP 当前 `refs/heads/main` 的 `cmds/dumpstate` 目录。

主要文件：

- `frameworks/native/cmds/dumpstate/main.cpp`
- `frameworks/native/cmds/dumpstate/dumpstate.cpp`
- `frameworks/native/cmds/dumpstate/dumpstate.h`
- `frameworks/native/cmds/dumpstate/DumpstateService.cpp`
- `frameworks/native/cmds/dumpstate/DumpstateInternal.cpp`
- `frameworks/native/cmds/dumpstate/DumpstateUtil.cpp`
- `frameworks/native/cmds/dumpstate/DumpPool.h`
- `frameworks/native/cmds/dumpstate/TaskQueue.h`
- `frameworks/native/cmds/dumpstate/dumpstate.rc`
- `frameworks/native/cmds/dumpstate/README.md`

对应线上源码：

- <https://android.googlesource.com/platform/frameworks/native/+/refs/heads/main/cmds/dumpstate/main.cpp>
- <https://android.googlesource.com/platform/frameworks/native/+/refs/heads/main/cmds/dumpstate/dumpstate.cpp>
- <https://android.googlesource.com/platform/frameworks/native/+/refs/heads/main/cmds/dumpstate/dumpstate.h>
- <https://android.googlesource.com/platform/frameworks/native/+/refs/heads/main/cmds/dumpstate/DumpstateService.cpp>
- <https://android.googlesource.com/platform/frameworks/native/+/refs/heads/main/cmds/dumpstate/DumpstateInternal.cpp>
- <https://android.googlesource.com/platform/frameworks/native/+/refs/heads/main/cmds/dumpstate/DumpstateUtil.cpp>
- <https://android.googlesource.com/platform/frameworks/native/+/refs/heads/main/cmds/dumpstate/DumpPool.h>
- <https://android.googlesource.com/platform/frameworks/native/+/refs/heads/main/cmds/dumpstate/TaskQueue.h>
- <https://android.googlesource.com/platform/frameworks/native/+/refs/heads/main/cmds/dumpstate/dumpstate.rc>
- <https://android.googlesource.com/platform/frameworks/native/+/refs/heads/main/cmds/dumpstate/README.md>

---

## 2. 它是什么

`dumpstate` 不是单纯的命令输出器，也不是某个 Binder service 的客户端包装器。

它的本质是：

`一个面向 bugreport 的总控编排器。`

它负责的事情包括：

- 接收 bugreport 请求
- 管理 bugreport mode / flag / 参数组合
- 组织命令执行、文件采集、`dumpsys` 采集、trace 截取
- 管理 root -> shell 的权限切换
- 处理进度、用户同意、取消、回调
- 生成临时文本、日志和最终 zip
- 在 API 场景下把产物复制回调用方 fd

因此它和 `dumpsys` 的定位差异很大：

- `dumpsys`
  - 是 Binder dump 客户端
- `dumpstate`
  - 是整机 bugreport 编排系统

`dumpsys` 只是 `dumpstate` 采集管线中的一个子步骤。

---

## 3. 目录与模块划分

### 3.1 `main.cpp`

入口很薄，只负责区分两种启动模式：

- 普通命令行模式
  - 直接走 `run_main(argc, argv)`
- `-w` 等待模式
  - 启动 `DumpstateService`
  - 然后 `joinThreadPool()` 等待外部通过 Binder 发起 bugreport

也就是说，`dumpstate` 既可以是普通程序，也可以作为短生命周期 Binder 服务启动器。

### 3.2 `dumpstate.h`

这是核心数据结构定义文件。

里面最重要的是：

- `Dumpstate`
  - 主 orchestrator
- `Dumpstate::DumpOptions`
  - 请求参数与运行模式
- `Progress`
  - 进度估计与持久化统计
- `DurationReporter`
  - 分段耗时记录
- `DumpData`
  - 采集文件句柄和元数据

从 `Dumpstate` 的类型定义可以看到：

- 它支持多种 `RunStatus`
  - `OK`
  - `HELP`
  - `INVALID_INPUT`
  - `ERROR`
  - `USER_CONSENT_DENIED`
  - `USER_CONSENT_TIMED_OUT`
- 它支持多种 bugreport mode
  - `FULL`
  - `INTERACTIVE`
  - `REMOTE`
  - `WEAR`
  - `TELEPHONY`
  - `WIFI`
  - `ONBOARDING`
  - `DEFAULT`

### 3.3 `dumpstate.cpp`

这是主业务文件。

主要负责：

- 解析参数
- 初始化运行环境
- 组织具体采集步骤
- 模式分流
- 进度与用户同意处理
- zip 收尾
- 最终复制和清理

### 3.4 `DumpstateService.cpp`

这是 Binder 服务入口层。

它暴露的接口包括：

- `preDumpUiData`
- `startBugreport`
- `cancelBugreport`
- `retrieveBugreport`

所以它不是自己采集数据，而是：

- 接受 API 请求
- 做并发 / 参数 / 所有权校验
- 把真正执行委托给 `Dumpstate`

### 3.5 `DumpstateInternal.cpp`

这里是更底层、更贴近系统权限和 fd 处理的内部能力。

典型职责：

- `DropRootUser()`
- dump 某个 fd 或文件到目标 fd
- root/shell 相关组与 capability 切换

### 3.6 `DumpstateUtil.cpp`

这里放的是命令执行和子进程工具逻辑。

核心是：

- fork/exec
- waitpid with timeout
- 子进程 stdout/stderr 管理
- 以较稳的方式运行系统命令

### 3.7 `DumpPool.h`

这是并行慢任务的线程池。

它的设计目标不是高吞吐通用线程池，而是：

- 让 `dumpstate` 把一些耗时采集并行化
- 同时把任务输出重定向到临时文件
- 避免多个任务同时往主 bugreport 文本里乱写

### 3.8 `TaskQueue.h`

这是串行任务队列。

它主要用于：

- 把“最终必须串行收口”的工作排队
- 例如 zip entry 添加、清理动作

### 3.9 `dumpstate.rc`

这是 init 胶水配置。

它定义了三个重要 service：

- `dumpstate /system/bin/dumpstate -s`
- `dumpstatez /system/bin/dumpstate -S`
- `bugreportd /system/bin/dumpstate -w`

因此 `dumpstate` 的实际运行形态不止一种，而是由 init 按不同入口参数拉起。

---

## 4. 启动形态

### 4.1 普通命令行模式

如果没有 `-w`：

- `main()` 直接走 `run_main(argc, argv)`
- `run_main()` 调 `ds.ParseCommandlineAndRun(argc, argv)`
- 根据 `RunStatus` 统一转成退出码

退出码模型比较清晰：

- `0`
  - 成功，或 `--help`
- `1`
  - 参数非法
- `2`
  - 运行错误、用户拒绝同意、用户同意超时

### 4.2 Binder 服务等待模式

如果带 `-w`：

- `main()` 先调用 `DumpstateService::Start()`
- 然后进入 Binder thread pool
- 自己不立刻执行 bugreport
- 等待别人调用 `startBugreport()`

这说明：

- `-w` 模式下 `dumpstate` 不是“执行器入口”
- 而是“等待被 API 驱动的服务容器”

### 4.3 init 启动模式

`dumpstate.rc` 说明了 Android 系统里常见的几种启动方式：

- `dumpstate -s`
  - 把 zip 结果写到 control socket
- `dumpstate -S`
  - 完成后把文件位置写到 control socket
- `dumpstate -w`
  - 作为 bugreport Binder 服务等待调用

所以 `dumpstate` 的调用方不一定是 shell 用户，也可能是：

- init
- system service
- BugreportManager/Binder API 调用方

---

## 5. 参数与 mode 逻辑

### 5.1 命令行参数

`DumpOptions::Initialize(argc, argv)` 解析的主要参数有：

- `-o`
  - 自定义输出目录
- `-s`
  - 通过 socket 输出 zip
- `-S`
  - 通过 socket 输出文件位置
- `-v`
  - 只打印 header
- `-q`
  - 关闭震动
- `-p`
  - 需要截图
- `-P`
  - 做进度更新
- `-R`
  - remote mode
- `-L`
  - limited mode
- `-w`
  - 启动 Binder 服务并等待

另外还有几个兼容 no-op 参数：

- `-V`
- `-d`
- `-z`

### 5.2 bugreport mode

API 路径不靠这些单字符参数决定主模式，而是通过 `BugreportMode` 传入。

`SetOptionsFromMode(...)` 会把 mode 展开成具体布尔选项。

例如：

- `TELEPHONY`
  - 走 telephony-only 采集
- `WIFI`
  - 走 wifi-only 采集
- `ONBOARDING`
  - onboarding-only
- `INTERACTIVE` / `WEAR` / `REMOTE`
  - 会联动截图、进度更新等行为

这说明 `mode` 不是仅用于展示的标签，而是：

`高层策略输入。`

### 5.3 flags

`Dumpstate` 还支持 API 级 flag：

- `BUGREPORT_USE_PREDUMPED_UI_DATA`
- `BUGREPORT_FLAG_DEFER_CONSENT`
- `BUGREPORT_FLAG_KEEP_BUGREPORT_ON_RETRIEVAL`

这些 flag 会影响：

- 是否复用预采的 UI 数据
- 是否延后用户同意
- retrieve 后是否保留原始 bugreport 文件

### 5.4 参数合法性校验

`ValidateOptions()` 会明确拒绝一些冲突组合，例如：

- 外部 bugreport fd 与 `stream_to_socket` 同时存在
- progress socket 与 stream socket 冲突
- remote mode 与 progress/socket 模式冲突

所以 `dumpstate` 不是“尽量容忍一切参数”，而是先把输出语义定清楚。

---

## 6. 核心对象模型

### 6.1 `Dumpstate`

整个程序围绕一个 `Dumpstate` 单例工作。

它持有的核心状态包括：

- 当前 bugreport `id`
- `options_`
- listener
- progress
- 临时路径
- 最终 zip 路径
- 日志路径
- trace / tombstone / ANR 等采集状态
- DumpPool / TaskQueue

这说明它不是无状态命令，而是：

`一次 bugreport 生命周期的状态机容器。`

### 6.2 `Progress`

`Progress` 的特点不是简单加一，而是：

- 有初始估计上限
- 会按历史运行结果持久化统计
- 下次运行可调整 max 估计

这意味着 progress 百分比不是拍脑袋写死的，而是：

- 经验值
- 持续校准

### 6.3 `DumpPool`

`DumpPool` 的关键设计点：

- 固定线程数
- 每个任务先写自己的临时文件
- 返回 future，最后再合并进 bugreport

它解决的是：

- 并行采集
- 但不破坏主输出顺序

### 6.4 `TaskQueue`

`TaskQueue` 是收尾串行化装置。

典型用途：

- zip entry 添加
- 延后执行的清理动作

这能避免：

- 多线程直接并发碰 zip writer
- 中途失败时遗漏清理

---

## 7. 主执行路径

### 7.1 总入口

普通命令行路径是：

1. `run_main(argc, argv)`
2. `ParseCommandlineAndRun(...)`
3. `Initialize()`
4. `Run(...)`
5. `RunInternal(...)`
6. `HandleRunStatus(...)`

其中：

- `Initialize()` 会先维护一个递增的 bugreport id
- `Run()` 只是薄封装
- 真正主体都在 `RunInternal()`

### 7.2 `RunInternal()` 的前置初始化

`RunInternal()` 先做这些准备：

- 打印和校验 options
- 把进程 nice 调到高优先级
- 尽量把自己从 OOM killer 保护出来
- 获取 wakelock
- 注册 `SIGPIPE` 忽略
- 处理 dry-run / strict-run 属性

然后准备输出控制面：

- 如有需要先开 control socket
- 创建内部输出文件和目录
- 建立主文本、log、zip 路径

### 7.3 stdout / stderr 重定向

这是整个实现里很关键的一点。

`RunInternal()` 会：

- 备份当前 `stdout`
- 备份当前 `stderr`
- 把 `stderr` 重定向到 `dumpstate_log.txt`
- 把 `stdout` 重定向到临时文本 `tmp_path_`

所以它不是一边跑一边直接把最终内容流式打到终端。

更准确地说：

- 采集阶段主要写临时文件
- 最终再收口成 zip

### 7.4 早期高价值采集

在进入主 mode 分流前，它会尽快拿一些“越早越有价值”的状态：

- `/proc/cmdline`
- 必要时截图
- system trace snapshot
- UI traces snapshot
- `RunDumpsysCritical()`

这里的思想非常明确：

`先冻结现场，再做重活。`

否则越晚采集，越容易被后续 dumpstate 自身活动污染。

### 7.5 用户同意前置点

在 UI-intensive 的早期步骤做完后，`RunInternal()` 会：

- 通知 listener：UI-intensive dumps 已完成
- 触发 `MaybeCheckUserConsent(...)`

如果是下面这些情况，则不会做 consent 流程：

- shell 触发
- 不是通过 API 调用
- 选择了 deferred consent
- 某些允许 consentless 的 onboarding 场景

否则会去找 `incidentcompanion`：

- 发起授权请求
- 等待用户确认

### 7.6 按 mode 分流

然后根据 options 分流到不同主路径：

- `telephony_only`
  - `DumpstateTelephonyOnly(...)`
- `wifi_only`
  - `DumpstateWifiOnly()`
- `limited_only`
  - `DumpstateLimitedOnly()`
- `onboarding_only`
  - `DumpstateOnboardingOnly()`
- 默认
  - `DumpstateDefaultAfterCritical()`

这一步之后，才进入真正的大规模采集。

---

## 8. 默认完整模式的工作流

### 8.1 先抓 logcat 和 traces

`DumpstateDefaultAfterCritical()` 开头先做：

- 第一轮 logcat
- 抓 Dalvik/native traces
- root 权限下读取 tombstone / ANR 相关 dump fd

这里的 traces 还会结合 `DumpPool` 并行执行。

### 8.2 root 阶段与 drop-root

默认完整路径里会先完成必须 root 才能拿到的内容，然后调用 `DropRootUser()`。

`DropRootUser()` 的语义不是简单 `setuid(shell)`，它还会：

- 设置 supplementary groups
- 保留必要 capability
- 尽量让后续采集以 `shell` 身份运行

这背后的意图是：

- 前期拿关键敏感数据
- 后期尽量缩小高权限执行窗口

### 8.3 大规模采集阶段

drop-root 之后会进入大量常规采集步骤。

包括但不限于：

- 文件采集
- 各类系统命令
- 大量 `dumpsys`
- board/vendor dump
- trace、tombstone、ANR、properties、network、storage、kernel、logcat

这一层的设计风格来自上游 README 里明确写出的理念：

`exec not link`

也就是：

- 尽量执行外部命令采集
- 而不是把太多逻辑直接 link 到 dumpstate 进程里

原因很现实：

- bugreport 常发生在系统已经部分损坏时
- dumpstate 需要尽量减少对“其他库必须正常工作”的依赖

### 8.4 并行与串行收口

慢任务会进入 `DumpPool` 并行跑。

但是：

- 输出先落临时文件
- 最终再由 `WaitForTask(...)` 或 zip entry 队列统一收口

这保证了：

- 有并行度
- 但最终 bugreport 不会乱序污染

---

## 9. 输出与打包模型

### 9.1 不是直接写最终 zip

`dumpstate` 的主文本先写 `tmp_path_`，日志写 `log_path_`。

之后 `FinalizeFile()` / `FinishZipFile()` 才把它们正式收进 zip。

### 9.2 `FinishZipFile()` 做什么

`FinishZipFile()` 会：

- 先跑完所有挂起的 zip entry 任务
- 把主文本加入 zip
- 写 `main_entry.txt`
- 把 `dumpstate_log.txt` 加入 zip
- 完成 `zip_writer_->Finish()`
- 关闭 zip writer
- 清理临时文件

所以 zip 不是一个附带输出，而是整个流程的正式终态之一。

### 9.3 API 场景下的复制

如果是通过 API 调用，而且不是 deferred consent：

- `RunInternal()` 结束后会尝试 `CopyBugreportIfUserConsented(...)`
- 成功时把 bugreport/screenshot 复制到调用方 fd
- 如果用户同意超时，则不复制，但可保留在内部目录后续 retrieve

这说明“生成 bugreport”和“把 bugreport 交付调用方”在架构上是两个阶段。

---

## 10. Binder 服务工作流

### 10.1 `preDumpUiData`

这是一个“预采 UI 数据”的前置接口。

典型逻辑：

- 确认当前没有其他 bugreport 在跑
- 取单例 `Dumpstate`
- 调 `PreDumpUiData()`

`PreDumpUiData()` 会尽早做：

- system trace snapshot
- UI trace snapshot

这样后续正式 `startBugreport()` 可以复用这批数据。

### 10.2 `startBugreport`

这是 Binder 服务层的主入口。

它会做这些事情：

1. 保证同一时刻只能有一个 bugreport 在跑
2. 校验 listener 是否存在
3. 校验 bugreport mode 合法
4. 初始化 `DumpOptions`
5. 校验传入的 bugreport fd / screenshot fd
6. 记录调用者 uid/package，供 cancel 校验
7. `Initialize()` 生成新的 dump id
8. 起独立线程执行真正 bugreport

所以服务层本身不做重活，只做：

- 守门
- 参数固化
- 生命周期管理

### 10.3 `cancelBugreport`

取消逻辑有所有权校验。

如果请求取消的人不是启动者：

- 返回 `EX_SECURITY`

如果是同一个调用者：

- 转调 `ds_->Cancel()`

因此取消不是“谁都能取消”，而是：

- 与 bugreport 所有权绑定

### 10.4 `retrieveBugreport`

这是“取回已生成 bugreport”的路径。

它会：

- 准备 `DumpOptions`
- 指定目标 bugreport 文件
- 创建独立线程执行 `Retrieve(...)`

这个接口对应的正是：

- 用户同意超时后稍后再取
- 或内部目录已存在 bugreport，需要后续交付

### 10.5 服务进程的退出模型

`DumpstateService.cpp` 的线程入口在完成后会直接 `exit(0)`。

这说明 `bugreportd` 不是常驻 daemon，而是：

- 一次请求
- 一次执行
- 执行完就退出

这也符合它在 `dumpstate.rc` 里被定义为 `oneshot` 的方式。

---

## 11. 权限与可靠性设计

### 11.1 高优先级和 OOM 保护

`RunInternal()` 会：

- 调高优先级
- 调整 `oom_score_adj` 或旧版 `oom_adj`

因为 bugreport 正常就是在异常环境里采集。

如果自己轻易被杀掉，整个工具就失去意义。

### 11.2 root -> shell 切换

`DropRootUser()` 会切换到 `shell`，同时尽量保留必要能力。

这表示上游设计不是“全程 root 跑完”，而是：

- 把 root 当作阶段性工具
- 不是默认执行身份

### 11.3 尽量少 link 更多 exec

README 里的“exec not link”原则非常重要。

它体现了 dumpstate 的容错哲学：

- 当系统有问题时，越少进程内耦合越安全
- 多执行外部工具，哪怕某个工具失败，也不必让 dumpstate 自己崩掉

### 11.4 输出隔离

通过：

- 临时文件
- `DumpPool`
- `TaskQueue`
- 最终 zip 收口

`dumpstate` 把“采集并发性”和“最终输出一致性”拆开处理。

这比所有任务直接并发写主 stdout 稳得多。

---

## 12. 它和 `dumpsys` 的关系

可以把几个工具这样区分：

- `cmd`
  - shell command 前端
- `service`
  - raw Binder transact 调试器
- `dumpsys`
  - Binder dump 聚合器
- `dumpstate`
  - bugreport 总控编排器

`dumpstate` 和 `dumpsys` 有重叠，但不是一回事。

重叠点：

- 都会调用系统 service
- `dumpstate` 内部会大量运行 `dumpsys`

本质差别：

- `dumpsys` 聚焦“取某些 service 的状态”
- `dumpstate` 聚焦“采一份完整系统问题现场并打包交付”

所以如果后续考虑 Go 重写：

- `dumpsys` 更像一个 CLI 工具重写问题
- `dumpstate` 更像一个系统级工作流编排问题

---

## 13. 如果未来要用 Go 重写，难点在哪里

### 13.1 重点不是 Binder 调用本身

真正难的不是 “Go 调一下 dumpsys/service”。

真正难的是：

- bugreport 生命周期
- 输出打包模型
- 权限切换
- 用户同意 / listener / cancel / retrieve
- 大量系统命令与临时文件编排

### 13.2 Android-only 属性更强

`dumpstate` 比 `dumpsys` 还更强依赖 Android 系统环境：

- init service
- Binder bugreport API
- `incidentcompanion`
- 持久化目录和权限模型
- wakelock
- zip 产物交付语义
- vendor/HAL 采集能力

因此如果重写，合理边界应当是：

- 明确只支持 Android
- 明确不追求非 Android 主机兼容

### 13.3 最适合的 Go 设计方向

如果未来实现 Go 版，不应机械照搬 C++ 文件级结构。

更自然的 Go 分层可能是：

- `cmd/dumpstate`
  - CLI 入口
- `internal/dumpstate/session`
  - 单次 bugreport 状态机
- `internal/dumpstate/collector`
  - 文件、命令、dumpsys、trace collector
- `internal/dumpstate/output`
  - 临时文件、zip、fd 交付
- `internal/dumpstate/service`
  - Binder 服务层
- `internal/dumpstate/runtime`
  - consent、progress、cancel、ownership

不过这已经不是“小工具移植”，而是：

`一个中等规模系统组件重写项目。`

---

## 14. 一句话结论

`dumpstate` 不是“更大的 dumpsys”，而是：

`一个围绕 bugreport 生命周期构建的系统级编排器，负责权限、采集、并行、同意、打包和交付。`
