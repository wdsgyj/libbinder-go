# AOSP `dumpsys` 程序工作流与逻辑分析

## 1. 上游源码范围

本次对齐的 AOSP 源码目录：

- `frameworks/native/cmds/dumpsys/main.cpp`
- `frameworks/native/cmds/dumpsys/dumpsys.cpp`
- `frameworks/native/cmds/dumpsys/dumpsys.h`
- `frameworks/native/cmds/dumpsys/Android.bp`

对应线上源码：

- <https://android.googlesource.com/platform/frameworks/native/+/refs/heads/main/cmds/dumpsys/main.cpp>
- <https://android.googlesource.com/platform/frameworks/native/+/refs/heads/main/cmds/dumpsys/dumpsys.cpp>
- <https://android.googlesource.com/platform/frameworks/native/+/refs/heads/main/cmds/dumpsys/dumpsys.h>
- <https://android.googlesource.com/platform/frameworks/native/+/refs/heads/main/cmds/dumpsys/Android.bp>

---

## 2. 它是什么

`dumpsys` 是 Android 系统里的通用 Binder dump 客户端。

它不是一个 service，也不是 service manager。

它的职责是：

1. 通过 `defaultServiceManager()` 获取当前系统 service 列表
2. 根据命令行参数筛选要处理的 service
3. 对目标 service 发起：
   - 常规 `dump`
   - 或 PID / stability / thread / clients 等调试型查询
4. 把结果输出到标准输出
5. 在客户端侧做超时控制和格式化包装

所以它的定位是：

- “统一系统诊断入口”
- 而不是某个单一 service 的专用工具

---

## 3. 构建产物

从 `Android.bp` 可以看到：

- `libdumpsys`
  - 静态库，便于测试和复用
- `dumpsys`
  - 系统侧可执行程序
- `dumpsys_vendor`
  - vendor 侧变体，binary 名仍然是 `dumpsys`

依赖包括：

- `libbinder`
- `libbinderdebug`
- `libserviceutils`
- `libbase`
- `libutils`
- `liblog`

这表明它除了常规 Binder 调用外，还依赖：

- Binder debug 能力
- `PriorityDumper` 相关公共工具

---

## 4. 入口流程

### 4.1 `main.cpp`

`main.cpp` 的逻辑非常薄：

1. 忽略 `SIGPIPE`
2. 取 `defaultServiceManager()`
3. 构造 `Dumpsys` 对象
4. 调 `dumpsys.main(argc, argv)`

如果默认 service manager 不存在，就直接输出：

```text
dumpsys: Unable to get default service manager!
```

然后返回 `20`。

### 4.2 `Dumpsys` 类

`dumpsys.h` 可以看出，`Dumpsys` 把流程拆成这些核心阶段：

- `main`
- `listServices`
- `setServiceArgs`
- `startDumpThread`
- `writeDump`
- `writeDumpHeader`
- `writeDumpFooter`
- `stopDumpThread`

所以它的设计不是“一个超长 main 函数”，而是比较清晰的：

- 参数解析
- service 列表构建
- 单个 service 执行
- 输出包装
- 超时/线程控制

---

## 5. 命令行模型

`dumpsys.cpp` 的 usage 里对外暴露的主要能力有：

- 无参数
  - dump 全部 service
- `-l`
  - 只列 service，不执行 dump
- `-t SECONDS`
  - 秒级超时
- `-T MILLISECONDS`
  - 毫秒级超时
- `--priority LEVEL`
  - 按 dump 优先级过滤
- `--proto`
  - 只保留支持 proto dump 的 service，并以 proto 方式请求 dump
- `--skip SERVICES`
  - 跳过指定 service
- `SERVICE [ARGS]`
  - 只 dump 某一个 service，并把 `ARGS` 透传给目标 service

另外还有几类“非标准正文 dump”模式：

- `--dump`
- `--pid`
- `--stability`
- `--thread`
- `--clients`

这些模式通过 `dumpTypeFlags` 组合控制。

如果用户没显式指定任何 dump type，默认就是：

- `TYPE_DUMP`

---

## 6. 参数解析与过滤逻辑

### 6.1 优先级

`--priority LEVEL` 只接受：

- `CRITICAL`
- `HIGH`
- `NORMAL`

内部会转成 `IServiceManager::DUMP_FLAG_PRIORITY_*`。

默认是：

- `DUMP_FLAG_PRIORITY_ALL`

### 6.2 proto 模式

`--proto` 做了两件事：

1. 只保留支持 proto dump 的 service
2. 给 service 的 dump 参数额外插入 `--proto`

另外，出于向后兼容考虑：

- 如果用户在 `SERVICE [ARGS]` 里自己传了 `--proto`
- `dumpsys` 也会把当前请求视为 proto dump

### 6.3 `--skip`

源码里的 `--skip` 行为有一个实际细节：

- 它会把后续位置参数逐个当成“要跳过的 service 名”
- 并没有在代码里做逗号拆分

也就是说，帮助文本虽然写了“comma-separated list”，但当前实现并不是严格按逗号拆分执行的。

### 6.4 单 service 与全量 service

如果：

- 没显式指定 service
- 或者用了 `-l`

那么 `dumpsys` 会先通过 `listServices(...)` 构造目标服务列表。

如果显式指定了某个 `SERVICE`：

- 就只处理它
- 并把剩余参数直接透传给该 service

---

## 7. service 列表构建

`Dumpsys::listServices(...)` 的逻辑是：

1. `sm_->listServices(priorityFilterFlags)`
2. 按名字排序
3. 如果 `filterByProto == true`
   - 再取一次 `sm_->listServices(DUMP_FLAG_PROTO)`
   - 计算两个列表交集

所以 proto 模式不是“先 dump 再失败”，而是：

- 先在 service manager 层筛掉不支持 proto 的服务

这能避免大量无意义请求。

---

## 8. service 参数重写

`setServiceArgs(...)` 会在真正发起 dump 前，给目标 service 自动补参数。

### 8.1 `--proto`

如果当前是 proto 模式：

- 在参数最前面插入 `--proto`

### 8.2 `-a`

如果当前是：

- dump 全部优先级
- dump `NORMAL`
- dump `DEFAULT`

就会在参数最前面插入：

- `-a`

这代表“dump all details”。

### 8.3 `--priority LEVEL`

如果当前明确只 dump：

- `CRITICAL`
- `HIGH`
- `NORMAL`

则会额外插入：

- `--priority`
- `LEVEL`

所以 `dumpsys` 并不是单纯调用 `IBinder::dump(fd, rawArgs)`，

而是会根据自身 CLI 语义对传给 service 的参数做一次统一改写。

---

## 9. 高层工作流

`Dumpsys::main()` 的整体工作流可以概括为：

1. 解析命令行
2. 生成目标 service 列表
3. 如果是多 service 或 `-l`
   - 先打印 `Currently running services:`
4. 对每个 service：
   - 如果被 skip，跳过
   - `startDumpThread(...)`
   - 可选写 header
   - `writeDump(...)`
   - 可选写 footer
   - `stopDumpThread(...)`

也就是说，它本质上是：

- 串行遍历 service
- 对每个 service 进行一次受控 dump

它不是并行 dump 全系统。

---

## 10. 为什么要用 thread + pipe + poll

这是 `dumpsys` 最核心的实现技巧。

### 10.1 问题背景

`service->dump(fd, args)` 是一个同步阻塞调用。

如果直接在主线程里执行：

- 主线程会卡住
- 客户端无法做超时控制
- 也无法边读边输出

### 10.2 实际做法

`startDumpThread(...)` 会：

1. 先 `checkService(serviceName)`
2. 创建一对 pipe
3. 把读端保存到 `redirectFd_`
4. 开一个线程
5. 在线程里把 dump 结果写到 pipe 的写端

然后主线程在 `writeDump(...)` 里：

- 对 pipe 读端做 `poll`
- 超时到了就返回 `TIMED_OUT`
- 有数据就 `read`
- 读到的数据立刻写到标准输出

### 10.3 这个设计带来的效果

它实现了三件事：

- 可以边 dump 边输出
- 可以在客户端侧实现超时
- 不会把主线程锁死在单个 service 的 Binder 调用里

### 10.4 限制

这里的 timeout 不是“强制终止远端 dump”。

实际上：

- 超时后主线程只是停止等待
- 并把 worker thread `detach`
- 远端 service 的 dump 逻辑未必会立刻停止

也就是说：

- `dumpsys` 能控制“自己等多久”
- 但不能真正取消远端正在执行的 dump

---

## 11. 单个 service 的执行路径

### 11.1 `startDumpThread`

它会先找目标 service。

找不到时：

- 输出 `Can't find service`
- 返回 `NAME_NOT_FOUND`

找到了以后：

- 创建 pipe
- 保存读端
- 起线程执行实际 dump

线程内部会根据 `dumpTypeFlags` 顺序执行：

- `TYPE_PID`
- `TYPE_STABILITY`
- `TYPE_THREAD`
- `TYPE_CLIENTS`
- `TYPE_DUMP`

这意味着：

- `PID / stability / thread / clients` 这些信息会先写
- 常规 `dump()` 往往作为最后的“正文”输出

### 11.2 `writeDump`

主线程的 `writeDump(...)` 负责：

- `poll`
- `read`
- `WriteFully`
- 统计 `bytesWritten`
- 计算 `elapsedDuration`
- 处理 timeout

如果 timeout 且不是 proto 模式：

- 还会补一条人类可读的超时提示

### 11.3 `stopDumpThread`

如果 dump 成功完成：

- `join`

如果超时或失败：

- `detach`

最后关闭 pipe 读端。

---

## 12. 各种 dump 类型的含义

### 12.1 `--dump`

最常见，也是默认行为：

- 调 `service->dump(fd, args)`

### 12.2 `--pid`

通过：

- `service->getDebugPid(&pid)`

输出服务宿主进程 PID。

### 12.3 `--stability`

通过：

- `internal::Stability::debugToString(service)`

输出 Binder stability 信息。

### 12.4 `--thread`

先取服务 PID，再通过 `binderdebug` 查询：

- 当前线程使用数
- 总线程数

输出类似：

```text
Threads in use: X/Y
```

### 12.5 `--clients`

通过 `binderdebug` 读取 Binder driver 里的 client PID 信息。

如果 service 是 local binder：

- 直接提示本地 binder 没法拿 client PID

如果是 remote binder：

- 用 Binder handle + service PID + current PID 查询 client 列表

---

## 13. 输出组织

### 13.1 列表输出

多 service 或 `-l` 时，先输出：

```text
Currently running services:
  activity
  package
  ...
```

### 13.2 header

多 service dump 时，会在每段前加 header。

普通情况：

```text
DUMP OF SERVICE activity:
```

指定优先级时：

```text
DUMP OF SERVICE CRITICAL activity:
```

### 13.3 footer

每段结束后会写：

- 本次 dump 用时
- 结束时刻

例如：

```text
--------- 0.123s was the duration of dumpsys activity, ending at: 2026-03-26 12:34:56
```

### 13.4 timeout 输出

如果某个 service 超时：

- `writeDump()` 里会追加 timeout 提示
- `main()` 在收到 `TIMED_OUT` 后又会再打印一段 timeout 提示

所以当前实现里 timeout 提示存在重复输出的可能。

这是源码层面一个比较明显但不影响主流程的细节。

---

## 14. 与 `cmd` / `service` / `servicemanager` 的关系

这几个程序都和 Binder service 打交道，但职责不同。

### 14.1 `dumpsys`

- 面向诊断
- 统一抓各 service dump 输出
- 自带超时控制和格式化包装

### 14.2 `cmd`

- 面向 shell command
- 调 `SHELL_COMMAND_TRANSACTION`
- 需要 `IShellCallback` / `IResultReceiver`

### 14.3 `service`

- 面向原始 transact 调试
- 需要手工知道 code 和参数布局

### 14.4 `servicemanager`

- 是这些工具背后的服务发现后端
- 实现 `IServiceManager`

所以可以理解为：

- `servicemanager` 提供名字服务
- `dumpsys` 做统一诊断输出
- `cmd` 做 shell-command 分发
- `service` 做低层 raw transact 调试

---

## 15. 对 Go 重写的启发

如果后续要在 Go 里实现 `dumpsys`，上游这份实现给出的关键启发是：

### 15.1 不能只做简单 `service.dump()`

还需要补齐：

- service 列表过滤
- priority/proto 参数重写
- header/footer 格式化
- timeout 与错误输出

### 15.2 timeout 需要客户端侧隔离

如果没有：

- worker thread / goroutine
- pipe / socket / buffer 重定向
- 主线程 poll/select

就没法做到“边输出边超时控制”。

### 15.3 `dump` 不是唯一能力

至少还要考虑：

- PID
- stability
- thread usage
- client PID

否则只能算“最小 dump wrapper”，而不是完整 `dumpsys`。

### 15.4 Android-only 场景更合理

这些能力强依赖：

- Binder driver
- `getDebugPid`
- `binderdebug`
- ServiceManager dump flags

所以它天然就是 Android-only 工具，不值得为了非 Android 环境做复杂兼容层。

---

## 16. 一句话结论

`dumpsys` 的本质不是“打印所有服务状态”的单一命令，而是：

`一个基于 ServiceManager 做服务发现、以 pipe + worker thread + poll 实现超时控制、再统一包装输出的 Binder 诊断前端。`
