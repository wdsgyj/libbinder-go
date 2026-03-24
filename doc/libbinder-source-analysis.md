# libbinder 源码逻辑梳理

## 1. 分析范围

- 源码位置: `aosp-src/frameworks/native/libs/binder`
- 参考版本: `platform/frameworks/native` `master`
- 提交: `4f463a6b1de9198963dc6aff74154a504ba3f8f6`
- 提交时间: `Tue Mar 25 17:41:28 2025 -0700`
- 来源记录: `aosp-src/frameworks/native/libs/binder.UPSTREAM`

这份 `libs/binder` 已经不只是“经典 C++ Binder 客户端/服务端库”，而是一个统一的 Binder 通信平台，包含：

1. 经典 kernel Binder 通道
2. 基于 socket 的 Binder RPC 通道
3. C++/NDK/Rust 三层接口
4. ServiceManager、lazy service、accessor 等服务治理能力
5. Trusty 和 host 场景的适配层

一句话概括它的架构：

`libbinder = 统一对象模型 + 两套传输后端 + 多语言绑定 + 平台服务治理`

## 2. 目录与逻辑模块映射

| 逻辑模块 | 主要目录/文件 | 作用 |
| --- | --- | --- |
| 对象模型与 C++ API | `include/binder/IBinder.h`, `Binder.h`, `BpBinder.h`, `IInterface.h`, `Binder.cpp`, `BpBinder.cpp` | 定义 Binder 抽象、typed interface、local stub、remote proxy |
| 内核 Binder 传输 | `ProcessState.cpp`, `IPCThreadState.cpp`, `include/binder/ProcessState.h`, `include/binder/IPCThreadState.h` | 管理 `/dev/binder`、线程池、ioctl 读写、命令分发 |
| 序列化与错误模型 | `Parcel.cpp`, `Status.cpp`, `Stability.cpp`, `RecordedTransaction.cpp` | 负责对象扁平化、reply/exception 编码、稳定性约束、事务录制 |
| 服务管理 | `IServiceManager.cpp`, `BackendUnifiedServiceManager.cpp`, `LazyServiceRegistrar.cpp`, `ServiceManagerHost.*` | 服务注册、查询、等待、缓存、lazy 生命周期管理 |
| Binder RPC | `RpcSession.cpp`, `RpcServer.cpp`, `RpcState.cpp`, `RpcTransport*.cpp`, `RpcTlsUtils.cpp` | 无内核 Binder 传输，基于 socket/TLS/Trusty |
| 平台专用接口 | `IActivityManager.cpp`, `IPermissionController.cpp`, `ProcessInfoService.cpp`, `aidl/android/...` | Android 平台侧的具体服务接口和 AIDL |
| NDK 接口层 | `ndk/ibinder.cpp`, `ndk/parcel.cpp`, `ndk/service_manager.cpp`, `ndk/status.cpp` | 向 C/NDK 暴露稳定 ABI |
| Rust 接口层 | `rust/src/*.rs`, `rust/sys/*` | 基于 NDK 封装 Rust Binder API |
| 适配与可移植层 | `OS_android.cpp`, `OS_non_android_linux.cpp`, `UtilsHost.cpp`, `liblog_stub`, `binder_sdk` | host 构建、非 Android Linux、SDK 打包等支持 |
| 测试与 fuzz | `tests/`, `tests/parcel_fuzzer`, `tests/rpc_fuzzer`, `tests/unit_fuzzers` | 单元测试、集成测试、性能测试、模糊测试 |
| Trusty 端口 | `trusty/`, `include_trusty/`, `RpcTransportTipcAndroid.cpp`, `RpcTrusty.cpp` | Trusty 环境下的 Binder 和 RPC 适配 |

## 3. 总体架构图

```text
Typed Interface / AIDL
    |
    v
BnInterface / BpInterface
    |
    v
BBinder <------> BpBinder
    |                |
    |                +-- 远端句柄可能是:
    |                    1) kernel binder handle
    |                    2) RPC session + rpc address
    |
    +-- onTransact()
          ^
          |
Parcel + Status + Stability
          |
          +-- 后端 A: IPCThreadState + ProcessState + /dev/binder
          |
          +-- 后端 B: RpcSession + RpcState + RpcServer + socket/TLS/Trusty
                      |
                      +-- ServiceManager / Accessor / LazyService
                      +-- NDK / Rust bindings
```

这套设计的关键点是：上层对象模型尽量统一，而传输层允许切换。`BpBinder` 和 `Parcel` 都已经显式支持“kernel Binder”和“RPC Binder”两种后端。

## 4. 主干模块分析

### 4.1 对象模型：IBinder / BBinder / BpBinder / IInterface

这是 `libbinder` 最核心的一层。

- `IBinder` 是统一抽象，定义事务码、`transact`、`linkToDeath`、`getExtension`、`setRpcClientDebug` 等低层协议。
- `BBinder` 是本地对象的基类，代表“服务端 stub 实体”。
- `BpBinder` 是远端对象代理，代表“客户端 proxy 实体”。
- `IInterface`、`BnInterface<T>`、`BpInterface<T>` 则是 typed interface 包装层，供手写接口或 AIDL 生成代码使用。

几个关键观察：

1. `IBinder` 的事务码除了业务调用区间外，还内建了一批元操作。
   例如 `PING_TRANSACTION`、`INTERFACE_TRANSACTION`、`DUMP_TRANSACTION`、`EXTENSION_TRANSACTION`、`SET_RPC_CLIENT_TRANSACTION`。
2. `BBinder::transact` 先处理这些内建事务，再把业务事务交给 `onTransact`。
3. `BpBinder` 不是“只会走内核 Binder 的代理”。它内部用 `std::variant` 保存句柄，既可以持有 kernel binder handle，也可以持有 `RpcSession + rpc address`。
4. `IInterface.h` 的宏体系本质上是在做“typed Binder 接口模板化”。
   `asInterface` 先尝试 `queryLocalInterface`，命中则直接拿本地对象，否则退化为创建 `BpInterface`。
5. 手写接口仍被支持，但头文件里已经明确表达了偏好 AIDL 自动生成，手写接口只保留 allowlist。

这层对应的源码阅读顺序建议是：

1. `include/binder/IBinder.h`
2. `include/binder/Binder.h`
3. `include/binder/BpBinder.h`
4. `include/binder/IInterface.h`
5. `Binder.cpp`
6. `BpBinder.cpp`

### 4.2 经典内核 Binder 调用链

经典同步调用的主干路径是：

```text
客户端 typed proxy
  -> BpBinder::transact
  -> IPCThreadState::transact
  -> writeTransactionData
  -> talkWithDriver (BINDER_WRITE_READ ioctl)
  -> 内核 binder driver
  -> 服务端线程收到 BR_TRANSACTION
  -> IPCThreadState::executeCommand
  -> BBinder::transact
  -> BBinder::onTransact / 具体 BnXXX::onTransact
  -> sendReply
  -> 内核 binder driver
  -> 客户端 waitForResponse
```

这条链路分成两个状态对象：

#### 4.2.1 `ProcessState`: 进程级状态

`ProcessState` 是“每进程一个”的内核 Binder 全局状态，职责包括：

- 打开 binder driver，并校验协议版本
- `mmap` 一段事务接收区
- 管理 handle 到 `BpBinder` 的映射
- 管理 Binder 线程池参数和线程命名
- 管理 context manager 获取和 handle 0 代理创建

重要设计点：

1. `ProcessState::self()` 是单例入口。
2. `initWithDriver` 允许在首次初始化前切换到 `/dev/vndbinder` 等其他 driver。
3. `getStrongProxyForHandle` 负责把内核 handle 映射为 `BpBinder`，并处理弱引用竞争。
4. `startThreadPool` 只是启动线程池，不等于业务线程都 ready；真正处理事务要靠 `IPCThreadState::joinThreadPool`。

#### 4.2.2 `IPCThreadState`: 线程级状态

`IPCThreadState` 是“每线程一个”的通信状态，职责包括：

- 维护线程本地 `mIn`/`mOut` Parcel 缓冲区
- 发起事务、等待回复、发送 reply
- 解析 driver 返回的 `BR_*` 命令
- 保存当前事务上下文，如 calling pid/uid/sid、StrictMode、WorkSource
- 管理 death notification、freeze notification、句柄引用计数

其中最重要的三个函数是：

- `transact`: 组包并发起调用
- `waitForResponse`: 等待 `BR_REPLY` 或其他返回命令
- `executeCommand`: 把 `BR_TRANSACTION`、`BR_DEAD_BINDER` 等命令分发给对应逻辑

`executeCommand` 里对 `BR_TRANSACTION` 的处理非常关键：

1. 用 driver 返回的 buffer 构造 `Parcel`
2. 保存并切换当前线程的 calling identity
3. 找到目标 `BBinder`
4. 执行 `BBinder::transact`
5. 若不是 oneway，则回写 reply
6. 恢复线程上下文

这就是 Binder 服务端“看上去像普通函数调用”的根本原因。

### 4.3 Parcel、Status、Stability：协议与约束层

这一层决定了 Binder 调用在线上到底怎么编码、解码和校验。

#### 4.3.1 `Parcel`: 双格式序列化容器

`Parcel` 在当前版本里已经不是单一格式：

- 对 kernel Binder，它维护 `kernelFields`，核心是原始数据区和对象偏移数组
- 对 RPC Binder，它维护 `rpcFields`，核心是对象位置表和额外 fd 列表

这意味着 `Parcel` 本身已经是“后端无关接口 + 后端相关存储”的设计。

关键点：

1. `markForBinder` 会根据目标 binder 是否是 RPC binder 来切换格式。
2. `writeInterfaceToken` 不只是写接口名；在 kernel Binder 模式下还会写 StrictMode、WorkSource 和分区 header。
3. `enforceInterface` 在服务端不仅做 descriptor 校验，还会恢复线程上下文里的 StrictMode 和 WorkSource。
4. `flattenBinder`/`unflattenBinder` 负责把 `IBinder` 编码成：
   - kernel 模式下的 `flat_binder_object`
   - RPC 模式下的 `type + rpc address`

一个很重要的边界约束是：

- 不能把 socket RPC binder 直接塞进 kernel Binder 事务
- 也不能把普通 kernel binder proxy 直接透传到 RPC 会话

源码是显式禁止这种跨传输后端透传的，因为那会把调用者无意中变成“代理进程”。

#### 4.3.2 `Status`: 业务异常与传输错误分离

`binder::Status` 不是简单包装 `status_t`，而是把错误分成两层：

- 低层传输错误，如 `DEAD_OBJECT`、`FAILED_TRANSACTION`
- 业务异常，如 `EX_ILLEGAL_ARGUMENT`、`EX_SERVICE_SPECIFIC`

因此一次 Binder 调用通常需要区分：

1. 事务有没有成功到达对端
2. 对端业务逻辑有没有抛异常

这也是 AIDL 生成代码会先看 `transact` 返回值，再读 `Status` 的原因。

#### 4.3.3 `Stability`: system/vendor/vintf 稳定性

`Stability` 是近几年 Binder 里很关键的一层。

它跟踪的不是字节级 wire format，而是“接口随版本演进的稳定等级”。

当前核心等级是：

- `VENDOR`
- `SYSTEM`
- `VINTF`

`BpBinder::transact` 在发起用户事务前会做稳定性检查：

- 当前 binder 的稳定性不满足调用上下文要求时，直接返回 `BAD_TYPE`
- `FLAG_PRIVATE_VENDOR` 会影响所需稳定性等级

这层机制的价值是：把分区边界约束提前到用户态，而不是让错误在更靠后的位置爆炸。

#### 4.3.4 `RecordedTransaction`: 调试与录制

`RecordedTransaction` 提供了事务录制能力：

- 记录接口名、code、flags、时间戳
- 记录 data parcel 和 reply parcel
- 使用分块格式和校验和持久化

这更像“离线重放/诊断基础设施”，不属于主执行路径，但对排查 Binder 问题很有价值。

### 4.4 ServiceManager 与服务生命周期

这部分不是传输协议本身，但它决定 Binder 服务如何被发现、注册和管理。

#### 4.4.1 `IServiceManager`: C++ 侧统一入口

`IServiceManager` 暴露的核心能力包括：

- `checkService`
- `getService`
- `waitForService`
- `addService`
- `listServices`
- `registerForNotifications`
- `isDeclared`

现代实现并不是直接手写一套 Binder stub/proxy，而是：

```text
IServiceManager (legacy C++ facade)
  -> CppBackendShim
  -> BackendUnifiedServiceManager
  -> AIDL IServiceManager
```

这说明它已经从“手工 C++ 接口”过渡为“C++ facade + AIDL backend”。

#### 4.4.2 `BackendUnifiedServiceManager`: 统一后端与缓存

这个文件是新版 ServiceManager 结构里非常重要的一块。它做了几件事：

1. 统一把 `getService/checkService` 结果转成 `os::Service`
2. 为一部分服务提供 client-side cache
3. 区分 lazy service 和非 lazy service
4. 能把 accessor 类型服务转成可连接的 Binder 服务

可以把它理解为：

`ServiceManager 的协议适配层 + 缓存层 + 访问器整合层`

#### 4.4.3 `LazyServiceRegistrar`: 空闲服务自动摘除

`LazyServiceRegistrar` 是服务生命周期管理的另一个重点。

它通过 `IClientCallback` 统计是否还有客户端连接：

- 有客户端时保持注册
- 没有客户端时尝试 `tryUnregisterService`
- 若条件允许，甚至推动进程退出

这相当于给 Binder 服务加了一层“按需存活”机制，减少常驻资源占用。

### 4.5 Binder RPC：第二套传输后端

这是当前版本里最值得注意的演进方向之一。

#### 4.5.1 设计定位

Binder RPC 不是把经典 Binder 全部替换掉，而是在不依赖 kernel binder driver 的情况下，复用 Binder 的对象模型和事务语义。

它的主要组成是：

- `RpcSession`: 一个客户端和服务端之间的会话
- `RpcServer`: 监听 socket，创建 session
- `RpcState`: 维护 binder 与 rpc address 的映射、引用计数和命令流转
- `RpcTransport*`: 底层传输实现，如 raw socket、TLS、TIPC/Trusty

#### 4.5.2 `RpcSession`: 会话与连接池

`RpcSession` 不是“单连接对象”，而是“连接组”。

这样设计是为了支持：

- 并发同步调用
- nested Binder call
- callback 场景

关键点：

1. 它区分 incoming 和 outgoing 线程/连接。
2. `setMaxIncomingThreads` 和 `setMaxOutgoingConnections` 直接影响并发模型。
3. `transact` 先挑一个独占连接，再把事务交给 `RpcState`。
4. 当会话里的 binder 都释放后，会话可以整体 shutdown。

#### 4.5.3 `RpcServer`: 接入层

`RpcServer` 负责：

- 建立监听 socket
- accept 新连接
- 为连接创建 `RpcSession`
- 启动和关闭 join 循环

支持的入口包括：

- Unix domain socket
- bootstrap socket pair
- vsock
- inet
- raw socket fd

这使它既能用于设备内进程间，也能用于虚拟化、跨边界调试或特殊宿主环境。

#### 4.5.4 `RpcState`: 地址映射与远端对象管理

`RpcState` 是 RPC Binder 的核心状态机。

它负责：

- `onBinderLeaving`: 本地 binder 被发送出去时，为其分配 rpc address
- `onBinderEntering`: 收到远端 address 时，恢复为本地 `BpBinder`
- 维护 `timesSent/timesRecd`
- 在 session 结束时发送 obituary

最重要的设计约束是：

1. RPC 会话内的 binder 地址只在该会话中有意义
2. 不允许发送来自其他 RPC session 的 binder
3. 不允许把普通 kernel binder proxy 直接塞进 RPC

这保证了 RPC Binder 的语义边界是清晰的，不会悄悄退化成“链式代理”。

#### 4.5.5 统一对象模型的体现

经典 Binder 和 Binder RPC 能共存，靠的是两点：

1. `BpBinder` 内部统一抽象成“handle 或 rpc address”
2. `Parcel` 内部统一抽象成“kernelFields 或 rpcFields”

因此上层 typed interface、AIDL 生成代码、很多业务代码都不必知道自己跑在哪个后端上。

## 5. 扩展层：NDK、Rust、平台接口、host/Trusty

### 5.1 NDK 层

`ndk/` 目录把 C++ `libbinder` 包装成稳定 C ABI：

- `ibinder.cpp`: `AIBinder`、本地/远端 binder 封装、类关联、事务派发
- `parcel.cpp`: `AParcel`
- `service_manager.cpp`: `AServiceManager_*`
- `status.cpp`: NDK 状态模型
- `process.cpp`, `stability.cpp`, `binder_rpc.cpp`: 线程池、稳定性、RPC 相关 API

这个层的定位很明确：

- 给 NDK/AIDL 使用
- 给需要稳定 ABI 的模块使用
- 为上层 Rust 绑定提供基础

### 5.2 Rust 层

`rust/src/lib.rs` 顶部注释已经把定位说得很清楚：

- 这是 Android `libbinder` 的 safe Rust interface
- 主要服务于 Rust AIDL backend
- 底层依赖的是 binder NDK，因此可以用于 APEX 场景

可以把 Rust 层理解为：

`Rust 语言友好的 API 层`，而不是重新实现一套传输栈。

关键模块：

- `binder.rs`: Rust Binder 核心类型
- `proxy.rs`: 代理对象
- `parcel.rs`: Rust Parcel
- `service.rs`, `state.rs`: ServiceManager 与线程状态
- `binder_async.rs`, `binder_tokio/`: 异步适配
- `sys/`: bindgen 生成的低层 FFI

### 5.3 平台专用接口

在根目录还能看到不少看起来“不像 transport core”的文件，例如：

- `IActivityManager.cpp`
- `IPermissionController.cpp`
- `ProcessInfoService.cpp`
- `PermissionCache.cpp`
- `aidl/android/os/IServiceManager.aidl`
- `aidl/android/content/pm/*.aidl`

从 `Android.bp` 的注释也能看出，这部分更偏 Android 平台接口，历史上放在 `libbinder` 里，但并不属于 Binder 传输层本身。

阅读时最好把它们当成“平台接口层”看待，而不是 core binder runtime。

### 5.4 host / SDK / 可移植层

这部分容易被忽略，但对理解整个工程边界很重要：

- `OS_android.cpp`, `OS_non_android_linux.cpp`, `OS_unix_base.cpp`: OS 适配
- `UtilsHost.cpp`, `liblog_stub/`: host 构建支持
- `binder_sdk`、`libbinder_sdk`: 对外打包的 host SDK

说明 `libbinder` 现在不仅面向设备系统镜像，也考虑了 host 工具链和 SDK 输出。

### 5.5 Trusty 端口

`trusty/README.md` 明确说明这是一份 Trusty 版本的 `libbinder`。

也就是说，AOSP 把 Binder 语义继续带到了 Trusty 环境，相关实现散落在：

- `trusty/`
- `include_trusty/`
- `RpcTransportTipcAndroid.cpp`
- `RpcTrusty.cpp`

它和 RPC 传输层关系非常紧密。

## 6. 测试体系

`tests/` 目录覆盖面很广，可以反向帮助理解源码边界。

可以按主题分组：

- 经典 Binder 核心: `binderLibTest.cpp`, `binderBinderUnitTest.cpp`, `binderDriverInterfaceTest.cpp`
- Parcel/Status: `binderParcelUnitTest.cpp`, `binderStatusUnitTest.cpp`, `binderPersistableBundleTest.cpp`
- 稳定性: `binderStabilityTest.cpp`, `binderStabilityIntegrationTest.cpp`
- RPC: `binderRpcTest.cpp`, `binderRpcWireProtocolTest.cpp`, `binderRpcBenchmark.cpp`, `binderRpcUniversalTests.cpp`
- 录制/回放: `binderRecordReplayTest.cpp`, `binderRecordedTransactionTest.cpp`
- host/device: `binderHostDeviceTest.cpp`, `binderUtilsHostTest.cpp`
- fuzz: `parcel_fuzzer/`, `rpc_fuzzer/`, `unit_fuzzers/`

从测试分布能看出，当前工程最被重视的几个主题是：

1. Parcel 正确性
2. RPC 正确性和协议兼容性
3. 稳定性约束
4. 极端输入鲁棒性

## 7. 推荐阅读顺序

如果目标是快速吃透 `libbinder`，建议按下面顺序读，而不是从目录顶层顺着看：

1. `include/binder/IInterface.h`
   先理解 typed interface、`BnInterface`、`BpInterface`。
2. `include/binder/IBinder.h`
   理解统一抽象和元事务码。
3. `include/binder/Binder.h` + `include/binder/BpBinder.h`
   理解 local stub 和 remote proxy。
4. `Binder.cpp` + `BpBinder.cpp`
   看对象层的真实行为。
5. `include/binder/ProcessState.h` + `include/binder/IPCThreadState.h`
   搭起 kernel Binder 运行时模型。
6. `ProcessState.cpp` + `IPCThreadState.cpp`
   理解线程池、ioctl、命令分发。
7. `include/binder/Parcel.h` + `Parcel.cpp`
   理解序列化格式和 Binder/FD 的扁平化。
8. `include/binder/Status.h` + `Status.cpp`
   理解异常与传输错误的分层。
9. `include/binder/Stability.h` + `Stability.cpp`
   理解 system/vendor/vintf 约束。
10. `include/binder/IServiceManager.h` + `IServiceManager.cpp`
    看服务注册与发现。
11. `BackendUnifiedServiceManager.cpp` + `LazyServiceRegistrar.cpp`
    看现代服务治理逻辑。
12. `include/binder/RpcSession.h` + `include/binder/RpcServer.h`
    建立 RPC 心智模型。
13. `RpcSession.cpp` + `RpcServer.cpp` + `RpcState.cpp`
    看第二套传输后端如何复用对象模型。
14. `ndk/` 和 `rust/`
    最后再看语言绑定层。

## 8. 结论

`libbinder` 的源码可以归纳成三层主线：

1. 上层统一对象模型
   `IBinder/BBinder/BpBinder/IInterface`
2. 中层协议与运行时
   `Parcel/Status/Stability + ProcessState/IPCThreadState`
3. 下层多后端与扩展生态
   `kernel Binder + Binder RPC + NDK + Rust + Trusty + ServiceManager`

如果从工程演进角度看，这份源码最重要的变化不是“又多了几个接口文件”，而是它已经从传统的 kernel Binder C++ 库，演进成一个可跨传输后端、跨语言、跨运行环境复用的 Binder 平台。

对 `libbinder-go` 这类项目来说，真正值得优先对标的不是某个单独类，而是下面这几个稳定骨架：

- Binder 对象模型: `IBinder / BBinder / BpBinder`
- 调用路径: `BpBinder -> IPCThreadState/RpcSession -> BBinder`
- 序列化规则: `Parcel`
- 错误模型: `Status`
- 传输后端边界: kernel Binder 与 RPC Binder 的区分
- 服务治理: `IServiceManager / LazyServiceRegistrar`

只要这几条主线抓住了，再往上移植 typed interface、AIDL 生成层或多语言绑定，思路会清晰很多。
