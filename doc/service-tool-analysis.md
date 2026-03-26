# AOSP `service` 工具逻辑分析

## 1. 上游源码范围

本次对齐的 AOSP 源码目录：

- `frameworks/native/cmds/service/service.cpp`
- `frameworks/native/cmds/service/Android.bp`

对应线上源码：

- <https://android.googlesource.com/platform/frameworks/native/+/refs/heads/main/cmds/service/service.cpp>
- <https://android.googlesource.com/platform/frameworks/native/+/refs/heads/main/cmds/service/Android.bp>

---

## 2. 原版 `service` 的职责

`service` 是一个低层 Binder 命令行调试工具。

它的职责不是统一封装系统 service 的 shell command，而是：

1. 通过 `defaultServiceManager()` 发现 service
2. 列出 service 列表和 descriptor
3. 对指定 service 发起一笔原始 Binder transact
4. 按命令行参数手工构造 `Parcel`
5. 以文本形式打印 reply parcel

从 `Android.bp` 可以看到，上游实际会构建三个变体：

- `service`
- `vndservice`
- `aservice`

其中：

- `vndservice` 通过 `-DVENDORSERVICES` 切到 `/dev/vndbinder`
- `aservice` 是 host 侧构建

当前项目已经明确只考虑 Android 环境，因此这里只对齐 `service` 的 Android 语义，不扩展 host `aservice` 或非 Android RPC delegate 默认 service manager。

---

## 3. 原版命令模型

### 3.1 `service list`

行为：

- 调 `sm->listServices()`
- 对每个 service 再调用一次 `checkService`
- 额外打印 interface descriptor

输出形式类似：

```text
Found N services:
0    activity: [android.app.IActivityManager]
1    package: [android.content.pm.IPackageManager]
...
```

### 3.2 `service check SERVICE`

行为：

- 调 `sm->checkService(name)`
- 只报告 `found` 或 `not found`

输出形式类似：

```text
Service activity: found
```

### 3.3 `service call SERVICE CODE ...`

行为：

1. 先通过 `checkService` 找到目标 Binder
2. 取出目标 interface descriptor
3. `data.writeInterfaceToken(ifName)`
4. 按命令行参数逐项写入 `Parcel`
5. 发起 `service->transact(code, data, &reply)`
6. 打印 `Result: <reply>`

这里的 `call` 是“原始 transact 工具”，不是高层业务 API。

它要求调用者自己知道：

- transaction code
- 参数顺序
- 参数类型
- 目标 service 的 parcel 协议

另外，上游实现会先执行：

- `data.markForBinder(service)`

这样 `Parcel` 会根据目标 Binder 的传输后端决定内部格式。

在当前项目里已经明确只考虑 Android kernel Binder，因此 Go 版实现不需要再为非 Android / RPC delegate 场景保留这层切换逻辑。

---

## 4. 参数编码矩阵

上游 `service.cpp` 在 `call` 模式下支持这些参数：

| 参数 | 语义 |
| --- | --- |
| `i32 N` | 写入 `int32` |
| `i64 N` | 写入 `int64` |
| `f N` | 写入 `float32` |
| `d N` | 写入 `float64` |
| `s16 STR` | 写入 UTF-16 `String16` |
| `null` | 写入 `null` strong binder |
| `fd FILE` | 打开文件并写入 FD |
| `nfd FD` | 直接把数值型 FD 写入 parcel |
| `afd FILE` | 创建 ashmem/匿名共享内存 FD，写入文件内容 |
| `intent ...` | 按固定布局写入一个 Intent 参数块 |

### 4.1 `intent` 的写入布局

上游实现的 `intent` 不是通用 `Parcelable<Intent>` 编码器，而是工具内部固定拼出来的一段 parcel 数据。

写入顺序是：

1. `action`
2. `data`
3. `type`
4. `launchFlags`
5. `component`
6. `categories[]`
7. `extras = null`

支持的键：

- `action=...`
- `data=...`
- `type=...`
- `launchFlags=...`
- `component=...`
- `categories=a,b,c`

这也是为什么 `service call ... intent ...` 只能算一个工具特化能力，而不是通用 Intent 编码层。

补充一点：

- 上游源码里 `intent` 分支是真实存在的
- 但 usage 文本中这段帮助被注释掉了，没有对外展示

---

## 5. 与 `cmd` 的关系

`service` 和 `cmd` 都会：

- 通过 `IServiceManager` 找 service
- 和系统 service 交互
- 提供命令行入口

但两者职责明显不同。

### 5.1 `service`

定位：

- 低层 Binder 原始调试工具

特点：

- 手工指定 transaction code
- 手工拼 request parcel
- 直接打印 reply parcel
- 不理解 shell command 协议

### 5.2 `cmd`

定位：

- 高层 shell-command 前端

特点：

- 面向实现了 `shellCommand` 的系统 service
- 自动传入 `stdin/stdout/stderr`
- 提供 `IShellCallback`
- 提供 `IResultReceiver`
- 更适合 `activity` / `package` / `window` 这类系统命令

### 5.3 重叠与边界

重叠点：

- 都能列 service
- 都依赖 service manager

非重叠点：

- `service` 适合“我知道 transact code 和参数协议，要手工打包”
- `cmd` 适合“我要调用 service 暴露出来的 shell command”

因此它们不是互相替代关系，而是两个不同层级的工具。

---

## 6. Go 版实现映射

当前仓库对应实现目录：

- `cmd/service`

关键文件：

- `cmd/service/main.go`
- `cmd/service/run.go`
- `cmd/service/run_test.go`

### 6.1 入口

- `Main(ctx, argv, stdout, stderr) int`
  - 打开默认 Binder 连接
  - 获取 `ServiceManager`
  - 转交到 `Run`

- `Run(ctx, argv, opts) int`
  - 负责命令解析和调度
  - 可在测试里注入 fake `ServiceManager`

### 6.2 子命令映射

- `list`
  - 对应 `runList`
- `check`
  - 对应 `runCheck`
- `call`
  - 对应 `runCall`

### 6.3 `call` 的参数编码

Go 版已经覆盖上游主参数矩阵：

- `i32`
- `i64`
- `f`
- `d`
- `s16`
- `null`
- `fd`
- `nfd`
- `afd`
- `intent`

实现映射：

- 基础标量直接写 `binder.Parcel`
- `fd` 通过打开文件后写 `WriteFileDescriptor`
- `nfd` 直接把提供的数值型 FD 封装为 `FileDescriptor`
- `afd` 在当前实现里使用匿名临时文件承载内容

说明：

- 上游 C++ 使用 `ashmem_create_region`
- 当前 Go 项目明确只考虑 Android，但仍保持纯 Go 路线，没有额外引入 ashmem 封装
- 因此这里采用“匿名临时文件 FD”来表达“单独承载文件内容的共享 FD”语义
- 对 `service` 这种调试工具来说，这个语义已经足够覆盖使用目的

### 6.4 reply 输出

上游 C++ 直接依赖 `TextOutput` 打印 `Parcel`。

Go 版当前做法是：

- 打印 payload size
- 打印十六进制数据
- 打印 Binder object table
- 对 object 打印：
  - kind
  - offset
  - length
  - handle
  - stability

这比简单打印 `[]byte` 更适合排查 Binder 事务问题。

---

## 7. 测试策略

为了让 `cmd/service` 可以稳定回归，测试没有依赖真实 `/dev/binder`。

而是通过 fake `ServiceManager` 和 fake `Binder` 覆盖这些点：

1. 帮助输出
2. `list` 输出格式
3. `check` 的 `found / not found`
4. `call` 的参数编码矩阵
5. `fd / nfd / afd` 的 FD 传递
6. `intent` 固定布局编码
7. reply dump 输出
8. 非法参数与 usage 分支

除此之外，已经做了 Android 真机验证：

- `service list`
- `service check activity`

都可以在设备上正常运行。

---

## 8. 当前结论

Go 版 `cmd/service` 当前已经具备一套完整可用的 Android Binder 调试工具能力：

- 能列 service
- 能检查 service 是否存在
- 能手工发起原始 Binder transact
- 能覆盖上游核心参数矩阵
- 有自动化单测和设备侧实测

它和 `cmd` 的关系应理解为：

- `cmd` 是 shell-command 前端
- `service` 是原始 Binder transact 工具

两者都需要保留，而且职责边界很清晰。
