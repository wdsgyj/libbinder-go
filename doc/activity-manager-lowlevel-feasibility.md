# ActivityManager Low-Level Feasibility

## 结论

按照 Android Java client 侧的分层方式，在 `service/lowlevel` 建一个 Go 版 `ActivityManager` facade 是可行的。

当前已经可行的部分是：

- 用 `activity` service 对齐 `android.app.ActivityManager.getService()`
- 用 `activity_task` service 对齐 `android.app.ActivityManager.getTaskService()` / `android.app.ActivityTaskManager.getService()`
- 用懒加载缓存对齐 Java 里的 `Singleton`
- 把 shell-command helper 与真正的 Binder typed client 分层隔离

当前还 **不适合直接做成 Java 等价的完整功能集**。主要不是 facade 层的问题，而是它背后的 AIDL 接口面和 Android framework parcelable 依赖图非常大。

## Java 层实际分层

我在 2026-03-27 检查了 AOSP `main` 分支的下面几个官方文件：

- `ActivityManager.java`
- `ActivityTaskManager.java`
- `IActivityManager.aidl`
- `IActivityTaskManager.aidl`

源码入口：

- <https://android.googlesource.com/platform/frameworks/base/+/refs/heads/main/core/java/android/app/ActivityManager.java>
- <https://android.googlesource.com/platform/frameworks/base/+/refs/heads/main/core/java/android/app/ActivityTaskManager.java>
- <https://android.googlesource.com/platform/frameworks/base/+/refs/heads/main/core/java/android/app/IActivityManager.aidl>
- <https://android.googlesource.com/platform/frameworks/base/+/refs/heads/main/core/java/android/app/IActivityTaskManager.aidl>

Java facade 的核心模式其实很薄：

1. 通过 `ServiceManager.getService(...)` 找到 Binder
2. 用 `IActivityManager.Stub.asInterface(...)` 或 `IActivityTaskManager.Stub.asInterface(...)` 包成 typed proxy
3. 用 `Singleton` 做进程内缓存
4. 在 facade 层补少量参数默认值、异常转换和便捷方法

因此，Go 侧在 `service/lowlevel` 做一个同风格 facade 并不难。

## 真正的难点

### 1. 接口面很大

按当前 `main` 分支粗略统计：

- `IActivityManager.aidl` 约 `306` 个方法签名
- `IActivityTaskManager.aidl` 约 `97` 个方法签名

这意味着即使 facade 很薄，底下也需要非常大的 typed proxy / stub / parcel codec 面。

### 2. 依赖大量 Android framework 类型

这两个 AIDL 直接 import 了大量 framework 类型，包括但不限于：

- `Intent`
- `Bundle`
- `ComponentName`
- `IntentFilter`
- `ProfilerInfo`
- `WaitResult`
- `ParceledListSlice`
- `ApplicationErrorReport`
- `ActivityManager.RunningTaskInfo`
- `ActivityManager.RecentTaskInfo`
- `Configuration`
- `Rect`
- `Point`
- `Bitmap`
- `RemoteCallback`
- `IIntentReceiver`
- `IIntentSender`
- `IApplicationThread`

而当前仓库里，真正可用于 Binder typed client 的 Android framework parcelable / interface 实现基本还没有。现有 `service/intent.go` 只是 shell command 参数构造器，不是 `android.content.Intent` 的 Parcel codec。

### 3. 很多类型不是“自动生成完就能用”

即使 `aidlgen` 可以把 AIDL interface 生成出来，下面这些仍然需要手工补齐：

- custom parcelable sidecar 映射
- 各 parcelable 的 wire layout codec
- 回调接口的本地 Binder 注册
- 一些 framework 语义默认值和兼容行为

这部分工作量会远大于 facade 本身。

## 与当前仓库能力的关系

当前仓库已经具备的基础：

- Binder transact/runtime
- `aidlgen` 的 interface/proxy/stub 生成能力
- custom parcelable sidecar 机制
- shell-command helper 基础设施

因此路线不是“做不到”，而是“不能一步到位”。

更准确的说法是：

- `service/lowlevel` facade：现在就能做
- `IActivityTaskManager` / `IActivityManager` 的完整 typed Go client：需要分阶段补 framework 类型族

## 建议路线

### 阶段 1

先固定 `service/lowlevel` 的 Java 风格 facade：

- `LookupActivityService`
- `LookupActivityTaskService`
- `LookupActivityManager`
- `ActivityManagerProvider`

这一步已经落地。

### 阶段 2

优先补最小可用 typed 子集，而不是一次性追完整个 `IActivityManager`：

- `Intent`
- `Bundle`
- `ComponentName`
- `ProfilerInfo`
- `WaitResult`
- `IIntentReceiver`
- `IIntentSender`

### 阶段 3

先生成并接入 `IActivityTaskManager` 的启动活动相关子集：

- `startActivity`
- `startActivityAsUser`
- `startActivityAndWait`
- `startActivities`

这是最接近现有 `cmd activity start-activity` 场景的一层。

### 阶段 4

再扩展 `IActivityManager` 中与 service / broadcast / app state 强相关的子集：

- `startService`
- `stopService`
- `broadcastIntentWithFeature`
- `registerReceiver`

## 当前落地形态

现在仓库里的分层建议是：

- `service/`
  - 保留 shell-command helper
- `service/lowlevel/`
  - 作为 Java 风格 facade 和未来 typed Binder client 的入口

这比继续把所有能力都塞进 `cmd activity` 风格 helper 更接近 Android Java client 的真实分层，也更适合作为后续生成式 AIDL 接口接入点。
