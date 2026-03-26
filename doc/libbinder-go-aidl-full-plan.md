# AIDL 到 Go Client/Server 全功能实现计划

## 1. 文档目标

这份文档定义一个明确目标：

- 最终实现“根据 AIDL 文件生成 Go client 和 server 代码”
- 目标不是只支持一个很小的 demo 子集
- 目标是尽量覆盖 AIDL 语言与 Binder 运行时在 Go backend 上应有的完整能力

这里的“全功能”指的是：

- 面向 Go backend 的 AIDL 语言能力完整
- 生成代码能落到当前基于 kernel Binder driver 的 Go 用户态 runtime 上
- 不是生成 Java / CPP / NDK / Rust backend
- 不是重做 Android Soong 的完整 `aidl_interface` 构建系统
- 不是用 Go 重写 kernel Binder

简单说，本计划的目标是：

`做一个可生产使用的 AIDL -> Go backend，建立在现有 Go Binder runtime 之上。`

当前状态：

- `0.0.6` 已完成阶段 0 到阶段 9 的全部交付
- parser / resolve / typed model / codegen / CLI / host / Android emulator 回归都已落地
- Go runtime 路线图阶段 11 也已完成，因此当前剩余工作只属于计划之外的后续增强，而不是本计划中的主链缺口

---

## 2. “功能全集” 的定义

如果只说“支持 AIDL”，边界会很模糊。这里先把目标拆清楚。

### 2.1 AIDL 语言前端能力

生成器最终应支持解析并建模：

- `package`
- `import`
- `interface`
- `oneway interface`
- 方法声明
- `in` / `out` / `inout`
- 常量
- `parcelable`
- structured parcelable
- 非 structured parcelable 声明
- `enum`
- `union`
- nested type declarations
- 动态数组 `T[]`
- 固定大小数组 `T[N]`
- `List<T>`
- `@nullable`
- `@nullable(heap=true)` 的字段语义
- `@VintfStability`
- `@FixedSize`
- `@Backing(type=...)`
- backend-specific annotation 的解析与保留

其中 backend-specific annotation 的策略要分两类：

- 对 Go backend 真有语义的，要实现
- 对 Go backend 是 no-op 的，也要能解析、保留并给出正确诊断，而不是直接炸掉

### 2.2 Go backend 类型能力

结合 AIDL 官方类型定义，Go backend 最终应覆盖：

- `boolean`
- `byte`
- `char`
- `int`
- `long`
- `float`
- `double`
- `String`
- `IBinder`
- interface type
- `T[]`
- `byte[]`
- `List<T>`
- `FileDescriptor`
- `ParcelFileDescriptor`
- parcelable type
- union type
- fixed-size array `T[N]`

### 2.3 生成代码层能力

生成器最终应能输出：

- Go interface
- proxy client
- stub / server dispatch
- transaction code 常量
- descriptor 常量
- `ServiceManager` 注册 / 查找 helper
- parcelable / enum / union 对应的 Go 类型
- 版本 / hash / stable AIDL 相关辅助代码

### 2.4 运行时语义能力

仅能“生成文件”不够，生成代码最终还必须依赖 runtime 真正跑起来。

因此 runtime 层最终必须支撑：

- 同步事务
- oneway 事务
- Binder interface 作为参数和返回值
- callback interface
- `in` / `out` / `inout`
- 结构化 parcelable 读写
- 自定义 parcelable 读写接入
- enum / union 读写
- `List<T>` / `T[]` / `T[N]`
- `FileDescriptor` / `ParcelFileDescriptor`
- death notification
- caller identity / 事务上下文
- stable AIDL 的 interface version / hash
- `UNKNOWN_TRANSACTION` / 旧版本回退语义

---

## 3. 当前状态

当前仓库已经完成了 Go Binder runtime 的 MVP 主链路：

- `/dev/binder` 通信底座
- 同步 / oneway 事务
- 本地服务注册与分发
- `ServiceManager`
- death notification
- 生命周期与显式 `Close`
- Android aarch64 模拟器测试脚本

这意味着：

- 代码生成器已经有可以挂接的 runtime
- 但 runtime 仍然只覆盖了 AIDL 全量能力中的一部分

### 3.1 当前已具备的基础能力

当前已经稳定具备的 `Parcel` / Binder 基础能力主要是：

- `int32`
- `uint32`
- `int64`
- `uint64`
- `bool`
- `String`
- nullable `String`
- `[]byte`
- 基础 Binder handle 读
- 本地 Binder object 写
- reply/status 基础处理

### 3.2 当前明确缺失的关键能力

当前还没有：

- AIDL parser / AST / import resolver
- codegen IR
- `float` / `double` / `char` 编解码
- 通用数组 `T[]`
- fixed-size array `T[N]`
- `List<T>`
- 通用 interface 参数写入
- callback interface 参数/返回值通路
- `FileDescriptor` / `ParcelFileDescriptor`
- structured parcelable
- custom parcelable adapter
- enum
- union
- nested type 生成
- `in` / `out` / `inout` 代码生成
- interface version / hash
- stable AIDL 校验与生成策略
- golden 级 codegen 测试体系

所以，当前离“全功能 AIDL 生成器”还差的不是模板层小修小补，而是：

1. AIDL 前端
2. Parcel 类型系统
3. runtime 语义补齐
4. stable AIDL 兼容层
5. 测试与兼容性体系

---

## 4. 主要缺口分层

为了避免后面规划失焦，先把缺口按层拆开。

### 4.1 第一层：AIDL 前端缺口

缺失内容：

- lexer / parser
- AST
- import graph
- 类型解析
- 注解解析
- 常量表达式解析
- nested type 解析
- 位置与诊断系统

没有这一层，就谈不上真正的代码生成器。

### 4.2 第二层：Go backend 类型系统缺口

缺失内容：

- AIDL type -> Go type 的完整映射
- nullability 的 Go 表达
- `in` / `out` / `inout` 的 Go 签名映射
- interface / callback 的 Go 表达
- custom parcelable 的 Go 接入模型
- union 的 Go 表达
- fixed-size array 的 Go 表达

这一层决定生成代码是不是“像 Go”，也决定它是不是可维护。

### 4.3 第三层：Parcel / runtime 缺口

缺失内容：

- 更完整的基础类型编解码
- 复合类型编解码
- Binder object 完整传输
- FD 传输
- `out` / `inout` 语义
- caller info
- stable AIDL 运行时语义

这是最关键的一层。因为如果 runtime 不支持，生成器写出来也只是空壳。

### 4.4 第四层：stable AIDL 缺口

缺失内容：

- interface version / hash 生成
- stable 类型约束校验
- `UNKNOWN_TRANSACTION` 兼容处理策略
- 新老版本 parcelable / enum / union 的回退行为设计

如果不补这层，生成器只能叫“普通 AIDL 样板生成器”，还不能叫“完整 AIDL backend”。

### 4.5 第五层：测试与兼容性缺口

缺失内容：

- parser 单测
- type checker 单测
- codec 单测
- golden codegen 测试
- 基于 AIDL 样例的端到端测试
- Android 模拟器中的 generated server/client 回归
- 版本兼容测试

没有这层，后续扩类型几乎一定会反复回归。

---

## 5. 设计原则

### 5.1 先补 runtime，再写生成模板

每新增一种 AIDL 能力，都必须先补：

1. type model
2. parcel codec
3. runtime semantics
4. codegen
5. tests

不能反过来。

### 5.2 每种类型都要贯穿五层

例如加 `List<T>`，不能只改 parser。

必须同时完成：

- 语法解析
- 语义校验
- Go 类型映射
- Parcel 编解码
- proxy/stub 生成
- 单测和集成测试

### 5.3 structured 与 custom parcelable 必须分开

这两个概念不能混成一个实现。

- structured parcelable：
  - 由 AIDL 直接定义字段
  - 生成器可以自动生成 Go struct 和 codec
- custom / unstructured parcelable：
  - AIDL 只有声明，没有字段布局
  - 需要 Go 侧显式提供 codec 适配

### 5.4 Parse all, fail only when necessary

对于很多 annotation：

- 可以先做到“能解析、能保留、能诊断”
- 不一定一上来就都对 Go backend 有行为

但不能因为 annotation 不认识就直接无法处理整个文件。

### 5.5 先做通用 backend，再做便利 helper

优先级应该是：

1. 核心 AIDL backend 正确
2. proxy/stub 可用
3. 再补 `RegisterService` / `WaitService` 等便利辅助

---

## 6. 关键技术决策，必须先定稿

在进入实现前，下面这些点必须冻结。

### 6.1 AIDL 基础类型的 Go 映射

必须明确：

- `byte` 映射为 `int8` 还是自定义别名
- `char` 映射为 `uint16` 还是更高层类型
- `String?` 与 `@nullable String` 如何表达
- `List<T>` 与 `T[]` 是否都映射为 `[]T`
- `T[N]` 是否映射为 `[N]T`

### 6.2 `out` / `inout` 的 Go API 设计

必须明确：

- `out` 参数用返回值还是指针参数
- `inout` 参数如何避免让生成代码变成 C 风格 API

建议：

- `out` 优先映射为额外返回值
- `inout` 映射为输入参数 + 返回更新值

不要直接照搬 Java/CPP 风格。

### 6.3 interface / callback 的 Go 表达

必须明确：

- 远端接口参数如何写入 `Parcel`
- 返回接口如何包装为 generated proxy
- 本地 callback 如何自动注册为 local Binder node

### 6.4 custom parcelable 的接入方式

AIDL 自身没有 Go backend 的 `go_type` 语法。

因此必须设计一个 Go backend 自己的外部映射方案，例如：

- sidecar 配置文件
- 代码注册表
- generator flag 指定的映射关系

否则自定义 parcelable 永远做不完整。

### 6.5 stable AIDL 的 Go 表达

必须明确：

- interface version / hash 是否生成为保留方法
- Go server 如何暴露自身版本 / hash
- Go client 如何做版本查询和旧版本回退

---

## 7. 分阶段实施计划

下面的顺序不是“建议”，而是推荐的真正实现顺序。

### 阶段 0：冻结支持矩阵与 Go backend 规范

#### 目标

先把 Go backend 的语言映射与边界定死。

#### 包含内容

1. 列出完整 AIDL feature matrix
2. 为每种类型定义 Go 映射
3. 定义 `in/out/inout` 的 Go API 形式
4. 定义 custom parcelable 接入方式
5. 定义 stable AIDL 在 Go backend 的行为

#### 输出物

- 一份 support matrix
- 一份 Go backend mapping spec
- 一份 custom parcelable 适配规范

#### 完成标准

- 任意一个 AIDL 构造都能回答“Go 里长什么样”

---

### 阶段 1：AIDL 前端实现

#### 目标

建立完整的解析和语义分析基础。

#### 包含内容

1. lexer
2. parser
3. AST
4. import 解析
5. 名字解析
6. annotation 解析
7. 常量表达式解析
8. nested type 解析
9. 诊断与 source position

#### 输出物

- `internal/aidl/parser`
- `internal/aidl/ast`
- `internal/aidl/resolve`
- 语法与解析单测

#### 完成标准

- 能稳定解析 AIDL 示例文件并得到可用于 codegen 的 AST

---

### 阶段 2：中间表示与 Go backend type model

#### 目标

把 AIDL AST 转成适合 Go 生成器使用的 typed IR。

#### 包含内容

1. type-checked IR
2. Go 类型映射
3. directionality 归一化
4. nullability model
5. default value model
6. interface / parcelable / enum / union / nested type IR

#### 输出物

- `internal/aidl/ir`
- `internal/aidl/gomodel`
- 类型系统单测

#### 完成标准

- 生成器不再直接依赖 AST，而只依赖 typed IR

---

### 阶段 3：基础 Parcel 类型补齐

#### 目标

先补齐所有标量和基础集合类型。

#### 包含内容

1. `byte`
2. `char`
3. `float`
4. `double`
5. `T[]`
6. `byte[]`
7. `List<T>`
8. fixed-size array `T[N]`
9. nullability
10. 基础默认值语义

#### 输出物

- `binder.Parcel` 新 codec
- 单元测试
- 针对生成器的 codec helper

#### 完成标准

- 基础类型已经不再是 codegen 阻塞项

---

### 阶段 4：Binder object 与 FD 传输补齐

#### 目标

把 interface type、callback、FD 通路补齐。

#### 包含内容

1. 通用 remote Binder 参数写入
2. interface type 返回值解码
3. callback interface 注册与回传
4. `IBinder` 支持
5. `FileDescriptor`
6. `ParcelFileDescriptor`
7. death/watch 与 generated proxy 的协同

#### 输出物

- Binder object 完整参数通路
- FD 通路
- 相关单测和 Android 集成测试

#### 完成标准

- AIDL interface 参数与 callback 不再是阻塞项

---

### 阶段 5：structured parcelable / enum / union / nested type

#### 目标

补齐用户定义类型。

#### 包含内容

1. structured parcelable 代码生成
2. structured parcelable codec
3. enum 与 backing type
4. union 的 tagged model 与 codec
5. nested type 生成
6. default values
7. `@FixedSize` 约束校验

#### 输出物

- 生成的 Go struct / enum / union
- codec 测试
- round-trip 测试

#### 完成标准

- 典型 AIDL 用户定义类型都可以自动生成并跑通

---

### 阶段 6：custom parcelable 与扩展适配

#### 目标

补齐 AIDL 中无法从源文件直接推导布局的类型。

#### 包含内容

1. custom parcelable registry
2. 生成器读取 sidecar 映射
3. Go codec 接口定义
4. 生成器与 registry 连接
5. 对未注册 custom parcelable 给出明确诊断

#### 输出物

- custom parcelable 适配机制
- 示例与文档

#### 完成标准

- `parcelable Foo;` 这类声明在 Go backend 中有完整接入路径

---

### 阶段 7：proxy / stub / service helper 生成

#### 目标

真正做出可用的生成器主体。

#### 包含内容

1. interface 常量生成
2. transaction code 生成
3. client proxy 生成
4. server stub 生成
5. `oneway` 生成
6. `in/out/inout` 生成
7. `CheckService` / `WaitService` / `AddService` helper
8. callback interface 生成

#### 输出物

- `cmd/aidlgen`
- 生成文件模板
- golden tests

#### 完成标准

- 从 AIDL 直接生成并编译通过一套 Go client/server 代码

---

### 阶段 8：stable AIDL 支持

#### 目标

补齐 stable AIDL 在 Go backend 上的生成与运行时行为。

#### 包含内容

1. `@VintfStability` 校验
2. interface version / hash 生成
3. generated proxy 查询 remote version / hash
4. generated stub 暴露本地 version / hash
5. 新旧版本兼容策略
6. `UNKNOWN_TRANSACTION` 回退策略
7. 新增字段 / 枚举 / union 分支的兼容规则

#### 输出物

- stable AIDL helper
- 兼容性测试

#### 完成标准

- Go backend 可以用于 stable AIDL 场景，而不是只支持非稳定接口

---

### 阶段 9：工具化与大规模验证

#### 目标

把生成器从“能跑”提升到“可维护、可回归、可扩展”。

#### 包含内容

1. 多文件 package 生成
2. import / output layout 管理
3. golden test corpus
4. AOSP AIDL 样本回归
5. Android 模拟器 generated e2e
6. 错误信息与诊断优化
7. 生成结果稳定性控制

#### 输出物

- 完整测试矩阵
- 示例仓库或 demo
- 回归脚本

#### 完成标准

- 新增 AIDL 特性时能快速判断是否破坏既有生成结果

---

## 8. 推荐执行顺序

必须遵守下面的顺序：

1. 阶段 0
2. 阶段 1
3. 阶段 2
4. 阶段 3
5. 阶段 4
6. 阶段 5
7. 阶段 6
8. 阶段 7
9. 阶段 8
10. 阶段 9

一句话就是：

`先定义 Go backend 规范 -> 再做 AIDL 前端 -> 再补 runtime/codec -> 最后做 generator 和 stable AIDL。`

不能倒过来。

---

## 9. 每一阶段都必须交付的内容

每一阶段至少要交付：

1. `code`
2. `test`
3. `doc`

更具体一点：

- `code`
  - parser / runtime / generator 的实际实现
- `test`
  - unit test
  - golden test
  - Android integration test（适用时）
- `doc`
  - 当前阶段做了什么
  - 还没做什么
  - 下一阶段依赖什么

---

## 10. 完成判定标准

只有当下面这些条件同时满足，才可以说“已经实现 AIDL 到 Go 的功能全集目标”：

1. AIDL 主语言构造都能解析并进入 typed IR
2. AIDL 官方主类型在 Go backend 都有明确映射
3. generated proxy/stub 可以覆盖同步、oneway、callback、`in/out/inout`
4. structured parcelable、custom parcelable、enum、union、nested type 都能用
5. interface / `IBinder` / FD / `ParcelFileDescriptor` 都能传
6. stable AIDL 的 version/hash 和兼容语义已实现
7. 有 parser、codec、generator、runtime、Android e2e 的自动化测试
8. 能拿一组真实 AIDL 文件生成 Go 代码并在 Android 环境里跑通

---

## 11. 当前最先要做的具体落点

如果从今天开始实施，最先应该落地的不是模板，而是下面四件事：

1. 写 support matrix，冻结 Go backend 的类型映射和 `in/out/inout` 设计
2. 建立 `internal/aidl/{ast,parser,resolve,ir}` 目录和最小解析器骨架
3. 补 `Parcel` 的基础类型缺口：
   - `byte`
   - `char`
   - `float`
   - `double`
   - `T[]`
   - `List<T>`
4. 定义 custom parcelable 的 Go codec 接口与 sidecar 映射方案

这四件事做完之后，才适合开始写第一版 `cmd/aidlgen`。

---

## 12. 参考边界

下面这些官方资料定义了本计划中的“功能全集”边界：

- AIDL language
  - https://source.android.com/docs/core/architecture/aidl/aidl-language
- AIDL backends
  - https://source.android.com/docs/core/architecture/aidl/aidl-backends
- AIDL annotations
  - https://source.android.com/docs/core/architecture/aidl/aidl-annotations
- Stable AIDL
  - https://source.android.com/docs/core/architecture/aidl/stable-aidl

这些资料特别影响了本计划中的以下判断：

- 需要覆盖 `in` / `out` / `inout`
- 需要覆盖 structured parcelable / enum / union / nested types
- 需要覆盖 `List<T>`、`T[]`、`T[N]`
- 需要处理 `@nullable`、`@FixedSize`、`@Backing`、`@VintfStability`
- 需要支持 interface version / hash 以及 stable AIDL 兼容行为

---

## 13. 一句话总结

要实现 AIDL 到 Go 的“功能全集”，真正的路径不是先做模板，而是：

`先补 AIDL 前端与完整类型系统，再补 Parcel/runtime，最后做生成器与 stable AIDL 兼容层。`
