# libbinder-go AIDL 支持矩阵

## 1. 目标

这份矩阵用于冻结 Go backend 的支持边界，并明确 `0.0.6` 的当前状态。

状态说明：

- `已具备基础实现`
  - 当前仓库已经有可运行代码或测试覆盖
- `已冻结规范`
  - 语义和 API 形式已经定稿，但实现还未完成
- `待实现`
  - 仍在后续阶段中

当前结论：

- `0.0.6` 已完成阶段 11 的增强能力收尾
- 当前 AIDL 全功能计划阶段 0 到阶段 9 已全部具备代码、测试与文档交付
- 当前 runtime 路线图阶段 1 到阶段 11 已全部具备自动化覆盖
- 当前不再存在既定路线图中的未完成阶段

---

## 2. AIDL 语言前端

| 能力 | 目标语义 | 当前状态 | 对应阶段 |
| --- | --- | --- | --- |
| `package` | 解析并保留 package 名 | 已具备基础实现 | 阶段 1 |
| `import` | 解析并保留 import 列表 | 已具备基础实现 | 阶段 1 |
| 多文件 import graph | 解析跨文件依赖与布局 | 已具备基础实现 | 阶段 1 / 9 |
| `interface` | 解析接口声明 | 已具备基础实现 | 阶段 1 |
| `oneway interface` | 解析 oneway 语义 | 已具备基础实现 | 阶段 1 |
| 方法声明 | 解析返回值、参数、名称 | 已具备基础实现 | 阶段 1 |
| `in` / `out` / `inout` | 解析方向信息并保留到 AST | 已具备基础实现 | 阶段 1 |
| `const` | 解析常量成员 | 已具备基础实现 | 阶段 1 |
| structured `parcelable` | 解析字段布局 | 已具备基础实现 | 阶段 1 |
| `parcelable` 内 `const` | 解析 structured parcelable 内部常量 | 已具备基础实现 | 阶段 1 / 5 |
| 非 structured `parcelable Foo;` | 解析为 opaque 类型声明 | 已具备基础实现 | 阶段 1 |
| `enum` | 解析枚举与可选显式值 | 已具备基础实现 | 阶段 1 |
| `union` | 解析 union 成员 | 已具备基础实现 | 阶段 1 |
| nested type | 解析 interface / parcelable 内嵌套声明 | 已具备基础实现 | 阶段 1 |
| `T[]` | 解析动态数组类型 | 已具备基础实现 | 阶段 1 |
| `T[N]` | 解析 fixed-size array 类型 | 已具备基础实现 | 阶段 1 |
| `List<T>` | 解析泛型容器 | 已具备基础实现 | 阶段 1 |
| annotation 语法 | 解析并保留 annotation 与参数 | 已具备基础实现 | 阶段 1 |
| annotation 语义校验 | 校验 `@nullable` / `@Backing` / `@FixedSize` / `@VintfStability` 等 | 已具备基础实现 | 阶段 1 / 8 |
| source position / 诊断 | 错误包含位置信息 | 已具备基础实现 | 阶段 1 |
| 常量表达式求值 | 支持移位、按位运算、符号引用与枚举自动取值 | 已具备基础实现 | 阶段 1 |
| field default value | 解析并保留 structured parcelable 字段默认值 | 已具备基础实现 | 阶段 1 / 5 |
| 名字解析 / 类型解析 | 建立 import 与符号表 | 已具备基础实现 | 阶段 1 / 2 |

---

## 3. Go backend 类型与 IR

| 能力 | 目标语义 | 当前状态 | 对应阶段 |
| --- | --- | --- | --- |
| 最小 AST | 支撑 parser 输出 | 已具备基础实现 | 阶段 1 |
| 最小 resolve | 基础重复声明诊断 | 已具备基础实现 | 阶段 1 |
| 最小 IR | 生成摘要级 IR | 已具备基础实现 | 阶段 2 |
| typed IR / Go model | 挂接类型检查与 codegen | 已具备基础实现 | 阶段 2 |
| AIDL -> Go 类型映射表 | 对全部官方类型给出稳定映射 | 已具备基础实现 | 阶段 0 / 2 |
| nullability model | 明确 nil / pointer / value 规则 | 已具备基础实现 | 阶段 0 / 2 |
| directionality model | 明确 `in/out/inout` 的 Go 签名 | 已具备基础实现 | 阶段 0 / 2 |
| nested type lowering | 将嵌套声明转为可生成命名空间 | 已具备基础实现 | 阶段 2 / 5 |
| const / default expression lowering | 将 AIDL 常量与默认值重写为 Go 表达式 | 已具备基础实现 | 阶段 2 / 5 |
| stable AIDL type model | version/hash 与兼容元数据进入 IR | 已具备基础实现 | 阶段 8 |

---

## 4. Parcel 与 runtime 基础类型

| 能力 | 目标语义 | 当前状态 | 对应阶段 |
| --- | --- | --- | --- |
| `boolean` | `bool` 编解码 | 已具备基础实现 | 已完成 |
| `byte` | `int8` 编解码 | 已具备基础实现 | 阶段 3 |
| `char` | `uint16` 编解码 | 已具备基础实现 | 阶段 3 |
| `int` / `long` | `int32` / `int64` 编解码 | 已具备基础实现 | 已完成 |
| `float` / `double` | `float32` / `float64` 编解码 | 已具备基础实现 | 阶段 3 |
| `String` | UTF-16 wire codec | 已具备基础实现 | 已完成 |
| `byte[]` | 可空字节数组 codec | 已具备基础实现 | 已完成 |
| `T[]` | 通用 slice helper | 已具备基础实现 | 阶段 3 |
| `List<T>` | 与动态数组共用 slice helper | 已具备基础实现 | 阶段 3 |
| `T[N]` | fixed-size helper + 长度校验 | 已具备基础实现 | 阶段 3 |
| nullable 集合 | `nil` 表示 null，空 slice 表示 empty | 已冻结规范 | 阶段 0 / 3 |
| `IBinder` | 通用 Binder object 传输 | 已具备基础实现 | 阶段 4 |
| interface type | typed proxy / stub / callback | 已具备基础实现 | 阶段 4 / 7 |
| `FileDescriptor` | FD 传输与所有权模型 | 已具备基础实现 | 阶段 4 |
| `ParcelFileDescriptor` | FD 包装与关闭语义 | 已具备基础实现 | 阶段 4 |
| structured parcelable codec | 自动字段编解码 | 已具备基础实现 | 阶段 5 |
| enum codec | backing type 与常量生成 | 已具备基础实现 | 阶段 5 |
| union codec | tag + payload 编解码 | 已具备基础实现 | 阶段 5 |
| custom parcelable codec | sidecar 适配与外部 codec 接入 | 已具备基础实现 | 阶段 6 |

---

## 5. 代码生成能力

| 能力 | 目标语义 | 当前状态 | 对应阶段 |
| --- | --- | --- | --- |
| `cmd/aidlgen` CLI | 读取 AIDL 并输出 AST/model/summary/Go 代码 | 已具备基础实现 | 阶段 7 的前置基础 |
| AST JSON 输出 | 调试 parser | 已具备基础实现 | 当前 |
| IR summary 输出 | 调试 lowering | 已具备基础实现 | 当前 |
| Go interface 生成 | 生成业务接口定义 | 已具备基础实现 | 阶段 7 |
| proxy client 生成 | 生成 typed client | 已具备基础实现 | 阶段 7 |
| stub / server 生成 | 生成 dispatch handler | 已具备基础实现 | 阶段 7 |
| transaction code 常量 | 自动分配并生成 | 已具备基础实现 | 阶段 7 |
| descriptor 常量 | 自动生成 | 已具备基础实现 | 阶段 7 |
| service helper | `Check/Wait/Add` typed helper | 已具备基础实现 | 阶段 7 |
| 多文件 package 输出布局 | `-out` 下保持 package 目录结构 | 已具备基础实现 | 阶段 9 |
| golden codegen corpus | 稳定输出回归 | 已具备基础实现 | 阶段 9 |

---

## 6. 稳定性与兼容能力

| 能力 | 目标语义 | 当前状态 | 对应阶段 |
| --- | --- | --- | --- |
| `@nullable` | 解析并映射到 Go nil 语义 | 已冻结规范 | 阶段 0 / 2 |
| `@nullable(heap=true)` | 解析并保留 heap 语义元数据 | 已具备基础实现 | 阶段 1 |
| `@FixedSize` | 解析并在 resolve/codegen 层校验 | 已具备基础实现 | 阶段 0 / 5 |
| `@Backing(type=...)` | 解析并影响 enum backing type | 已具备基础实现 | 阶段 0 / 5 |
| `@VintfStability` | 解析并进入 stable AIDL 语义 | 已具备基础实现 | 阶段 0 / 8 |
| interface version/hash | 保留事务码与缓存策略 | 已具备基础实现 | 阶段 8 |
| `UNKNOWN_TRANSACTION` 回退 | 旧版本兼容 | 已具备基础实现 | 阶段 8 |

---

## 7. 测试矩阵

| 能力 | 当前状态 | 备注 |
| --- | --- | --- |
| `Parcel` 单元测试 | 已具备基础实现 | 覆盖新增标量与集合 helper |
| parser 单元测试 | 已具备基础实现 | 覆盖 interface、nested type、annotation、数组 |
| resolve 单元测试 | 已具备基础实现 | 覆盖重复声明、annotation 语义、常量表达式 |
| IR 单元测试 | 已具备基础实现 | 当前覆盖最小 lowering |
| `aidlgen` CLI 单元测试 | 已具备基础实现 | 覆盖 AST / summary / go / import graph / AOSP corpus |
| Android runtime 集成测试 | 已具备基础实现 | 覆盖 Binder runtime 主链 |
| generated code e2e | 已具备基础实现 | 覆盖 host round-trip 与 Android checked-in generated fixture |
| AOSP AIDL corpus 回归 | 已具备基础实现 | host 侧覆盖 binder/aidl + binder/tests + parcel_fuzzer/parcelables |

---

## 8. `0.0.6` 结论

当前仓库已经具备完整 AIDL 代码生成与验证主链：

- parser / resolve / typed model / codegen / CLI / host e2e / emulator regression 都已打通
- Binder object / callback / FD / custom parcelable / stable AIDL 都已经进入生成结果
- import graph、多文件输出、golden corpus、AOSP binder corpus 回归都已建立
- checked-in generated fixture 已加入 Android aarch64 模拟器回归

因此：

- AIDL 全功能计划阶段 0 到阶段 9 已完成
- Go runtime 路线图阶段 1 到阶段 11 已完成
- 当前不再存在既定路线图中的未完成阶段
- 阶段 11 的增强能力已经进入主仓库：
  - Binder RPC backend
  - lazy service
  - record/replay
  - stability 标签
  - cache / debug snapshot / frame reuse
