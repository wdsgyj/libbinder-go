# AOSP `servicemanager` 程序逻辑分析

## 1. 上游源码范围

本次对齐的 AOSP 源码目录：

- `frameworks/native/cmds/servicemanager/main.cpp`
- `frameworks/native/cmds/servicemanager/ServiceManager.cpp`
- `frameworks/native/cmds/servicemanager/ServiceManager.h`
- `frameworks/native/cmds/servicemanager/Access.cpp`
- `frameworks/native/cmds/servicemanager/NameUtil.h`
- `frameworks/native/cmds/servicemanager/Android.bp`

对应线上源码：

- <https://android.googlesource.com/platform/frameworks/native/+/refs/heads/main/cmds/servicemanager/main.cpp>
- <https://android.googlesource.com/platform/frameworks/native/+/refs/heads/main/cmds/servicemanager/ServiceManager.cpp>
- <https://android.googlesource.com/platform/frameworks/native/+/refs/heads/main/cmds/servicemanager/ServiceManager.h>
- <https://android.googlesource.com/platform/frameworks/native/+/refs/heads/main/cmds/servicemanager/Access.cpp>
- <https://android.googlesource.com/platform/frameworks/native/+/refs/heads/main/cmds/servicemanager/NameUtil.h>
- <https://android.googlesource.com/platform/frameworks/native/+/refs/heads/main/cmds/servicemanager/Android.bp>

---

## 2. 它是什么

`cmds/servicemanager` 不是一个给用户手动敲命令的调试工具，而是 Android Binder 世界里的“服务注册中心”守护进程。

它的本体就是 `android.os.IServiceManager` 的服务端实现。

也就是说：

- `service list`
- `service check`
- `cmd -l`
- `defaultServiceManager()->checkService(...)`
- `addService(...)`

最终都是在和这个进程交互。

如果没有它：

- 进程无法通过名字找到 Binder service
- 新服务无法注册到全局命名表
- `waitForService` / registration notification / declared instance 查询都无从谈起

所以它不是普通客户端工具，而是 Binder 命名体系的核心基础设施。

---

## 3. 构建产物

从 `Android.bp` 可以看到，`cmds/servicemanager` 实际会构建多种变体：

- `servicemanager`
- `servicemanager.recovery`
- `vndservicemanager`
- 测试
  - `servicemanager_test`
  - `servicemanager_unittest`
- fuzz target
  - `servicemanager_fuzzer`

其中：

- `servicemanager`
  - 默认系统侧 service manager
- `servicemanager.recovery`
  - recovery 环境使用
- `vndservicemanager`
  - vendor Binder 命名空间对应的 service manager
  - 通过 `-DVENDORSERVICEMANAGER=1` 切换行为

这也说明它不是“一个简单 binary”，而是 Android 系统不同分区 / 环境下的核心基础组件。

---

## 4. 启动流程

`main.cpp` 的启动流程很清晰，核心步骤如下。

### 4.1 选择 Binder driver

- 默认使用 `/dev/binder`
- 如果命令行带参数，也可以显式指定 driver

这就是为什么 vendor 侧会有独立的 `vndservicemanager` 和对应 Binder 设备节点。

### 4.2 初始化 Binder 进程状态

- `ProcessState::initWithDriver(driver)`
- `setThreadPoolMaxThreadCount(0)`
- `setCallRestriction(ProcessState::CallRestriction::FATAL_IF_NOT_ONEWAY)`
- `IPCThreadState::self()->disableBackgroundScheduling(true)`

这里的含义是：

- `servicemanager` 不走常规 Binder thread pool
- 它采用 `Looper + binder polling` 的事件循环模型
- 并且限制调用模型，避免出现不符合预期的双向阻塞交互

### 4.3 构造 `ServiceManager` 实例并自注册

- `sp<ServiceManager> manager = sp<ServiceManager>::make(...)`
- `manager->setRequestingSid(true)`
- `manager->addService("manager", manager, ...)`

`setRequestingSid(true)` 很关键：

- 让 Binder 驱动把调用方 SID 传上来
- 这样 `Access` 才能做基于 SELinux SID 的权限判定

它还把自己注册成名为 `"manager"` 的服务，便于调试或内部访问。

### 4.4 成为 Binder context manager

- `IPCThreadState::self()->setTheContextObject(manager)`
- `ProcessState::becomeContextManager()`

这一步是整个进程最关键的职责切换：

- 句柄 `0` 背后的全局 Binder context manager 变成当前进程

之后所有 `defaultServiceManager()` 路径，本质上都会先打到这里。

### 4.5 进入 `Looper`

`main.cpp` 没有启动普通 Binder 线程池，而是：

- 创建 `Looper`
- 把 Binder FD 接到 `Looper`
- 再挂一个 `timerfd`
- 然后无限 `pollAll(-1)`

其中：

- Binder FD 用于接收 Binder 事务
- `timerfd` 每 5 秒触发一次，用来做 client callback 生命周期检查

---

## 5. 对外暴露的接口面

`ServiceManager.h` 表明这个进程实现的是 `os::BnServiceManager`，也就是 `IServiceManager` AIDL 服务端。

公开事务面包括：

- `getService`
- `getService2`
- `checkService`
- `checkService2`
- `addService`
- `listServices`
- `registerForNotifications`
- `unregisterForNotifications`
- `isDeclared`
- `getDeclaredInstances`
- `updatableViaApex`
- `getUpdatableNames`
- `getConnectionInfo`
- `registerClientCallback`
- `tryUnregisterService`
- `getServiceDebugInfo`

这说明它的职责早就不只是“名字 -> Binder”的 map。

它还承载了：

- registration notification
- declared instance 查询
- APEX / VINTF 元数据查询
- client lifecycle 回调
- debug 信息查询

---

## 6. 内部核心数据结构

`ServiceManager.h` 里的核心状态有三张表：

- `mNameToService`
  - `service name -> Service`
- `mNameToRegistrationCallback`
  - `service name -> IServiceCallback[]`
- `mNameToClientCallback`
  - `service name -> IClientCallback[]`

其中 `Service` 结构体保存：

- `binder`
- `allowIsolated`
- `dumpPriority`
- `hasClients`
- `guaranteeClient`
- `ctx`

`ctx` 又记录注册者的：

- `pid`
- `uid`
- `sid`

所以它维护的不只是 Binder 对象本身，还包括：

- 注册者身份
- isolated app 可见性
- dump 优先级
- lazy service 生命周期相关状态
- callback 订阅关系

---

## 7. 服务查找逻辑

查找链路主要在：

- `tryGetService`
- `tryGetBinder`

### 7.1 基础查找

查找时会：

1. 从 `mNameToService` 取条目
2. 对 isolated UID 做额外限制
3. 做 `canFind` SELinux 校验
4. 如果没找到且允许启动，则尝试拉起 lazy service

### 7.2 accessor 感知

`tryGetService` 不只是直接查服务名。

在非 vendor 变体下，它还会先尝试：

- `getVintfAccessorName(name)`

如果某个 AIDL 实例是通过 accessor 暴露的，它会把查找重定向到 accessor service。

这说明现代 `servicemanager` 已经认识到：

- “按实例直接拿 Binder”
- “先经 accessor 间接拿 Binder”

这两种访问模型。

### 7.3 lazy service 拉起

如果服务不存在且当前路径允许“找不到时启动”，则 `tryStartService` 会异步设置系统属性：

- `ctl.interface_start=aidl/<name>`

它的目标是触发 init 拉起对应的 lazy AIDL service。

这也是 `getService` 和 `checkService` 语义差异的重要来源之一：

- `getService`
  - 找不到时允许尝试启动
- `checkService`
  - 只做非阻塞查询

---

## 8. 服务注册逻辑

核心实现在 `addService`。

注册时会依次做这些约束：

1. 调用方不能是 app UID
   - `App UIDs cannot add services`
2. 做 `canAddService`
   - 走 SELinux `add` 权限校验
3. binder 不能为空
4. 服务名必须合法
   - 长度 `1..127`
   - 只允许字母、数字、`_`、`-`、`.`、`/`
5. 非 vendor 变体下做 declaration / stability 约束
6. 如有必要，对远端 Binder 建立 `linkToDeath`
7. 覆盖或写入 `mNameToService`
8. 触发 registration callback

### 8.1 为什么 app 不能随便注册服务

这是 Android 系统治理的一部分。

如果普通 app 能往全局 service manager 塞服务：

- 命名空间会失控
- 系统服务发现和权限边界会被破坏
- Treble / SELinux / service_contexts 都会失去约束意义

所以 `addService` 天然是系统域能力，不是通用应用能力。

### 8.2 重复注册处理

如果同名服务已经存在：

- 不会直接拒绝
- 而是记录日志并覆盖旧项

日志里会对比：

- 原注册者 UID
- SID
- PID

目的是帮助定位：

- 多实例安装
- 晚到的 death notification
- 异常重注册

---

## 9. 权限模型

权限逻辑集中在 `Access.cpp`。

### 9.1 调用方身份来源

`getCallingContext()` 从 Binder 调用上下文提取：

- `pid`
- `uid`
- `sid`

如果 Binder 没直接带 SID，就回退到 `getpidcon(pid)`。

### 9.2 校验动作

对外主要有三种权限检查：

- `canFind`
- `canAdd`
- `canList`

### 9.3 目标安全上下文来源

对于 `find/add`：

- 先通过 `selabel_lookup(..., service_contexts)` 找到目标 service name 对应的 target context
- 再执行 `selinux_check_access`

对于 `list`：

- 直接对 `service_manager` 类做 `list` 权限校验

所以一个常见现象是：

- Binder 协议本身没问题
- 但 `servicemanager` 仍然返回 `EX_SECURITY`

这通常是：

- `service_contexts` 没有对应项
- 或调用方 SID 没有 `find/add/list` 权限

---

## 10. VINTF、stability 与分区语义

这是现代 `servicemanager` 相比“早期名字服务”最关键的升级点。

### 10.1 declaration 检查

`addService` 在非 vendor 变体下会调用：

- `meetsDeclarationRequirements(ctx, binder, name)`

这段逻辑本质是：

- 如果 Binder stability 不要求 VINTF 声明，就放行
- 如果要求，就必须 `isVintfDeclared(name)`

也就是：

- 不是所有 Binder 都受 VINTF 约束
- 但声明了相应 stability 语义的服务，注册时必须能在 VINTF manifest 里找到

### 10.2 manifest 查询能力

`ServiceManager.cpp` 会读取：

- device manifest
- framework manifest
- recovery manifest

并提供这些查询：

- `isDeclared`
- `getDeclaredInstances`
- `updatableViaApex`
- `getUpdatableNames`
- `getConnectionInfo`
- accessor 解析

这些功能背后的含义是：

- service manager 不只是运行时注册表
- 它还是系统稳定性模型的一部分

### 10.3 名字格式

源码里能看到两套名字格式：

- native HAL
  - `{package}/{instance}`
- AIDL HAL
  - `{package}.{interface}/{instance}`

`NameUtil.h` 和 `AidlName`/`NativeName` 解析逻辑就是为此服务。

---

## 11. registration notification

`registerForNotifications` / `unregisterForNotifications` 维护的是：

- 某个服务名对应的 `IServiceCallback` 列表

语义上等价于：

- “服务出现时通知我”

实现要点：

- 注册时先做 `canFindService`
- isolated app 被直接拒绝
- callback 自身会 `linkToDeath`
- 如果服务已经存在，注册后立刻回调一次 `onRegistration`

这正是 `waitForService` 可以从轮询演进为通知驱动的基础。

---

## 12. client callback 与 lazy 生命周期

这部分是 `servicemanager` 最容易被忽略，但非常关键的能力。

### 12.1 `registerClientCallback`

允许服务端自己注册一个 `IClientCallback`，用于感知：

- 当前是否还有客户端持有该服务

约束很严格：

- callback 不能为空
- 调用方必须有 `canAddService`
- 服务必须存在
- 只能由服务自己为自己注册
  - 通过 calling PID 与服务注册 PID 比较
- 传入 binder 必须和当前已注册 binder 相同

### 12.2 `handleClientCallbacks`

`main.cpp` 里挂了一个每 5 秒触发的 `timerfd`。

每次触发时会：

- 遍历 `mNameToService`
- 调 `handleServiceClientCallback`
- 通过 Binder driver 的 strong ref count 判断是否还有客户端

### 12.3 `guaranteeClient`

这是一个很实用的抖动抑制状态位。

它的目的是：

- 某些查找 / 注册时，先暂时把“存在客户端”视为真
- 避免 lazy service 在边界时刻被过早判定为“无客户端可退出”

### 12.4 `tryUnregisterService`

允许服务尝试注销自己，但限制很多：

- 只能服务自己注销自己
- binder 必须匹配
- 如果 `guaranteeClient` 为真，拒绝
- 如果 driver 观测到还有客户端，拒绝

也就是说：

- `tryUnregisterService` 不是强制删除
- 它是“在确认没有客户端依赖时，允许服务安全退出”

这和 lazy service 的自动回收强相关。

---

## 13. death 处理

`ServiceManager` 自己实现了 `IBinder::DeathRecipient`。

当某个 service 或 callback 死掉时，`binderDied` 会：

- 从 `mNameToService` 移除死掉的 service
- 清理 registration callback
- 清理 client callback

这保证了内部索引不会长期保留失效 Binder。

---

## 14. 调试能力

`getServiceDebugInfo` 会返回：

- service name
- 注册该服务时记录的 debug PID

这对定位：

- 服务到底由哪个进程注册
- 多实例冲突
- 异常覆盖注册

很有帮助。

---

## 15. 和 `service` / `cmd` 的关系

三者职责完全不同：

### 15.1 `servicemanager`

- 系统守护进程
- `IServiceManager` 服务端
- 维护全局 Binder service registry

### 15.2 `service`

- 低层调试客户端
- 通过名字找到服务
- 手工发起原始 transact

### 15.3 `cmd`

- 高层 shell-command 客户端
- 面向实现了 `shellCommand` 的系统服务

所以关系是：

- `service` 和 `cmd` 都是客户端
- `servicemanager` 是它们背后的名字服务和治理中枢

---

## 16. 对 Go 重写的启发

如果要在 Go 用户态重写“client 侧 libbinder 能力”，`servicemanager` 给出的启发主要有三点。

### 16.1 `ServiceManager` 不能只做最小 map 包装

至少要考虑：

- list/check/add
- registration notification
- declared instance 查询
- debug info
- client callback / try unregister

### 16.2 stability 和 VINTF 不是附加特性

它们直接决定：

- 什么服务能注册
- 注册时用什么名字
- 哪些实例对调用方可见

所以 Go 版如果只做 “`map[string]Binder` + `CheckService`” 会丢掉 Android 现代语义的核心部分。

### 16.3 lazy service 生命周期需要治理接口配合

只做 `AddService`/`CheckService` 不够。

如果想逼近上游行为，还要有：

- registration notification
- client callback
- `tryUnregisterService`
- 对“是否存在客户端”的可观测能力

---

## 17. 结论

`cmds/servicemanager` 的本质不是“Binder 版 DNS”，而是：

- 全局服务名注册中心
- SELinux 权限闸门
- VINTF / stability / APEX 元数据查询入口
- lazy service 生命周期协调器
- `IServiceManager` 的完整服务端实现

所以在 Android Binder 体系里，它属于控制面核心组件，而不是普通工具程序。
