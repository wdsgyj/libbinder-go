# Java / libbinder / Go Binder 互操作说明

这份文档整理几个容易混淆、但对 `libbinder-go` 设计非常关键的结论：

- Java / C++ / NDK 为什么可以比较自然地互转 Binder
- 为什么当前 Go runtime 不能直接“接住” Java 已经持有的 `IBinder`
- 同一个远端 Binder 在不同 runtime 里，handle 应该怎样理解
- 如果要做 Java <-> Go 同进程互通，正确的桥接方向是什么

---

## 1. 先给结论

可以把当前局面概括成一句话：

> Go 和 Java/C++ 在内核 Binder 协议层是通的，但在进程内 Binder 对象层目前是不通的。

更展开一点：

1. `libbinder-go` 自己直接对接 `/dev/binder`，实现了一套独立的 Go 用户态 Binder runtime。
2. Java / C++ / NDK 共享 Android 官方的 `libbinder` / native Binder 对象模型。
3. 两边都使用同一个 kernel Binder driver，所以跨进程协议兼容。
4. 在同一个进程里，`libbinder` runtime 和 Go binder runtime 可以共存。
5. 但两边不共享同一个进程内 Binder 对象表示，因此不能在同进程里直接互相“接管”对方已经持有的 Binder 对象。

---

## 2. 分层理解

可以把整件事拆成两层：

### 2.1 协议层

这一层是：

- `/dev/binder`
- kernel Binder driver
- transaction buffer
- object offsets
- `flat_binder_object`

这一层对 Java、C++、NDK、Go 都是一致的。

只要 userspace 按同一套协议和内核驱动交互，跨进程就可以互通。

### 2.2 进程内对象层

这一层是：

- Java `IBinder` / `Binder` / `BinderProxy`
- C++ `sp<IBinder>` / `BBinder` / `BpBinder`
- NDK `AIBinder`
- Go 里的 `binder.Binder`、本地 handler、handle 表、线程状态、Parcel 结构

这一层不是天然统一的。

真正的问题几乎都出在这里，而不是出在内核协议层。

---

## 3. Java / C++ / NDK 为什么容易互通

原因不是“JNI 很神奇”，也不是“Java 代码本质就是 C++”。

真正原因是：

> Android 官方已经把 Java / C++ / NDK 都接到了同一套 native Binder 对象模型上。

### 3.1 Java Binder 本身就是 native 对象的壳

Java 层的：

- `android.os.Binder`
- `android.os.BinderProxy`

并不是纯 Java 实现。

它们底下都持有 native 指针：

- `aosp-src/frameworks/base/core/java/android/os/Binder.java`
- `aosp-src/frameworks/base/core/java/android/os/BinderProxy.java`

这意味着 Java `IBinder` 背后并不是另一套独立 Binder 语义，而是 native Binder 对象的 Java 包装。

### 3.2 NDK 的桥梁是 `sp<IBinder>`

NDK 暴露了两组关键转换：

- `AIBinder_fromJavaBinder`
- `AIBinder_toJavaBinder`

以及：

- `AIBinder_fromPlatformBinder`
- `AIBinder_toPlatformBinder`

相关实现位于：

- `aosp-src/frameworks/native/libs/binder/ndk/ibinder_jni.cpp`
- `aosp-src/frameworks/native/libs/binder/ndk/libbinder.cpp`

这套转换的本质不是“复制对象属性”，而是：

1. Java `IBinder` 先还原成 native `sp<IBinder>`
2. NDK `AIBinder` 再包装这个 `sp<IBinder>`
3. 反向同理，再从 `AIBinder` 取回 `sp<IBinder>`，生成 Java 对象

所以 Java / C++ / NDK 的互通基础是：

> 它们共享同一个 native `IBinder` 对象模型。

---

## 4. Go runtime 当前处在什么位置

`libbinder-go` 当前不是 `libbinder` 上的一层 Go 语法包装。

它做的是：

- 自己打开 `/dev/binder`
- 自己管理线程模型
- 自己维护 handle/address table
- 自己做 Parcel 编解码
- 自己维护本地 handler 与远端 proxy 抽象

也就是说：

> Go runtime 是另一套独立的 userspace Binder runtime。

因此，虽然它和 Java/C++ 都能跟 kernel Binder driver 说同一种语言，但它并不天然认识 Java/C++ runtime 里的那个进程内 Binder 对象。

### 4.1 可以共存，但默认不互通

需要特别强调：

> 在同一个进程里，官方 `libbinder` runtime 和 `libbinder-go` runtime 是可以共存的。

这不是冲突点。

真正的限制是：

- 它们是两套并列的 userspace runtime
- 默认不共享同一个进程内 Binder 对象层
- 因此“共存”不等于“对象可以直接互换”

换句话说：

- 同进程共存：可以
- 直接把一边已持有的 Binder 对象交给另一边原样继续用：当前不行

---

## 5. 当前问题的本质

当前真正的问题不是：

- Go 和 Java 的 Binder 协议不兼容

而是：

- Go runtime 没有接入 Android 官方那套 `IBinder` / `BpBinder` / `AIBinder` 对象体系

因此：

- Java 侧拿到一个 `IBinder`
- 不代表 Go 侧就能直接把这个 Java 对象变成当前 Go runtime 可用的 `binder.Binder`

这里缺的不是“协议文档”，而是：

- Java `IBinder`
- JNI / native bridge
- shared native Binder object
- Go runtime wrapper

这条桥。

---

## 6. 为什么不能直接复制一个 Go proxy

一个常见误解是：

> 如果 Java/C++ 已经拿到了某个 remote proxy，Go 能不能根据它的字段值，直接复制出一个等价 proxy？

答案是：

> 不能按“字段复制”的思路来理解。

原因有三点。

### 6.1 remote proxy 不是普通 struct

一个远端 proxy 背后依赖的不只是几个可见字段，还依赖 runtime 状态：

- 当前 runtime 的 handle 表
- 引用计数
- death recipient 状态
- attached object
- extension / stability 等附加状态
- runtime 内的缓存和单例约束

这些都不是“读几个属性再写回去”就能完整复刻的。

### 6.2 handle 不是全局真实身份

`BpBinder` 里的 handle 只是：

> 当前 Binder runtime / 当前 binder_proc 视角下，对某个远端 Binder 的局部编号。

它不是全局唯一 ID，也不是脱离 runtime 仍然成立的稳定句柄。

### 6.3 Android 官方互转也不是这么做的

Java / C++ / NDK 的互转方式是：

- 共享同一个底层 `IBinder`

不是：

- 按字段值复制一个新的 proxy

所以正确理解是：

- 能做的是 `wrap`
- 不能做的是 `clone`

---

## 7. handle 到底怎么理解

这个问题需要说得非常精确。

### 7.1 handle 是局部编号，不是全局编号

在 kernel Binder 里，远端 Binder 到达目标侧后，会在目标侧的引用表里得到一个 `desc/handle`。

这个 handle 的语义是：

> “在这个 Binder runtime 对应的 binder_proc 里，用哪个本地编号引用远端对象”

它不是“全系统唯一 Binder ID”。

这里还要补一个关键边界：

> 从内核侧看，Binder userspace 上下文是以一次 `open("/dev/binder")` 建立出来的 session/file 为起点的。

也就是说，内核并不是只看“Linux 进程号”来理解 userspace Binder 上下文，而是看：

- 哪个进程
- 打开了哪个 Binder driver fd
- 这个 fd 对应的那次 `open("/dev/binder")`

在讨论 handle、引用表和 runtime 是否共享时，这个边界非常重要。

### 7.2 同一个远端 Binder，在不同 runtime 里 handle 可能不同

如果同一个 Linux 进程里共存两套彼此独立的 Binder runtime，例如：

- C++ `libbinder` 自己打开 `/dev/binder`
- Go runtime 也自己打开 `/dev/binder`

那么它们可以在同一进程中共存，但通常会对应不同的 Binder userspace 上下文。

从内核视角看，这通常就意味着：

- 两边各自有自己的 `open("/dev/binder")`
- 各自有自己的 session/file
- 因而也各自维护自己的引用视图和 handle 空间

这时：

- 即使两边都指向同一个远端 Binder
- 两边看到的 handle 也不应被认为是同一个值

它们可能碰巧数值一样，也可能不一样，但这个值没有跨 runtime 的可比性。

### 7.3 只有共享同一个底层 runtime 时，handle 才能被视为同一个

如果两边共享的是同一个 native Binder runtime，比如都围绕同一个 `sp<IBinder>` / `BpBinder` 工作，那么看到的实际上就是同一个 proxy，对应 handle 也才有共同语义。

所以更准确的说法是：

> 同一个远端 Binder，不保证在不同独立 runtime 里映射成同一个 handle。

---

## 8. Intent / Bundle 传 Binder 能说明什么

`Intent` / `Bundle` 支持携带 `IBinder`，因此经由 AMS 传递 Binder 是成立的。

这能说明：

- Binder object 可以通过 framework 的 `Intent` / `Bundle` 路径跨进程传递

但这不等于：

- Go runtime 自动就能直接接住 Java 侧已经拿到的 `IBinder`

如果进程 B 中：

- Java 组件先收到了 `IBinder`
- Go 代码也在同一个进程里

那么当前项目下，Go 侧仍然没有现成办法直接把那个 Java `IBinder` 纳入自己的独立 runtime 对象体系。

---

## 9. 当前项目下可行与不可行的做法

### 9.1 直接把 Java `IBinder` 交给 Go runtime

当前不可行。

原因是 Go runtime 不共享 Java/libbinder 的 native 对象层。

### 9.2 让 Java 再通过一次 Binder 调用转交给 Go

当前可行，而且是最现实的办法。

流程是：

1. Java 收到 `IBinder`
2. Java 再调用一个 Go 暴露的 Binder/AIDL 服务
3. 把这个 Binder 当作参数传给 Go
4. Go runtime 在自己的世界里通过 Binder 驱动重新物化一个远端 proxy

这条路径的本质是：

> 不是“共享同一个对象”，而是“重新经 Binder 驱动创建等价远端引用”。

### 9.3 让 Java 持有 Binder，Go 通过 Java 代理调用

这也可行。

但这不属于“Go 真的拿到了 Binder”，而是：

- Go 借 Java 使用 Binder

### 9.4 建立 JNI / NDK / libbinder bridge

这是长期正确方向。

目标不是伪造 handle，也不是复制 proxy 字段，而是：

1. Java `IBinder` -> native `sp<IBinder>` / `AIBinder`
2. Go 包装这个共享的 native Binder 对象
3. 让 Go 和 Java 在同进程里围绕同一个底层 Binder 工作

---

## 10. 什么方向是错误的

下面这些方向不应该做。

### 10.1 把 goroutine id 当成 thread id 发给内核

不行。

kernel Binder 认的是发起 syscall 的真实 OS 线程，不接受用户指定的“虚拟线程 ID”。

### 10.2 伪造一个全局 Binder 整数 ID

不行。

Binder 的 handle 不是全局 ID，只在对应 runtime / `open("/dev/binder")` 建立出的局部上下文里有效。

### 10.3 复制 C++ proxy 的字段生成 Go proxy

不行。

这绕不过 runtime 内部状态、引用管理、handle 空间和缓存语义。

---

## 11. 对当前项目的设计启发

这组结论对 `libbinder-go` 有几个直接启发。

### 11.1 继续保持“协议兼容、对象层独立”的定位

当前 Go runtime 的价值就在于：

- 不依赖 `libbinder`
- 直接对接 kernel Binder driver
- 保持 Go-native 的实现方式

这条路线没有问题。

### 11.2 不要把“同进程直接互接 Java Binder”误判为纯协议问题

这不是 Parcel 格式或 driver 协议的问题。

这是：

- 进程内 Binder 对象模型桥接

的问题。

### 11.3 如果要补 Java <-> Go 同进程互通，应新增 bridge 层

正确方向应是：

- 新增一层可选 bridge
- 通过 JNI / NDK / `libbinder` 接住 Java `IBinder`
- 在 Go 里包装共享 native Binder 对象

而不是：

- 修改 kernel 协议
- 伪造 handle
- 复制 proxy 字段

---

## 12. 最后压缩成三句话

1. Java / C++ / NDK 之所以互转容易，是因为它们共享同一个 native Binder 对象模型。
2. `libbinder-go` 当前走的是另一套独立 userspace runtime，因此只能在协议层互通，不能直接接管 Java 已持有的进程内 Binder 对象。
3. 想要同进程互通，正确方向是 bridge 到共享 native Binder 对象，而不是伪造 thread id、伪造 handle 或复制 proxy 字段。
