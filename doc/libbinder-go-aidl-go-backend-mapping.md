# libbinder-go AIDL Go Backend 映射规范

## 1. 目标

这份文档冻结 AIDL 到 Go backend 的公开 API 形态。

目标不是复刻 Java / C++ backend 的表面形式，而是定义一个：

- 能映射全部 AIDL 主语言构造
- 与当前 `binder` runtime 兼容
- 符合 Go 风格的 generated API

---

## 2. 输出布局

### 2.1 目录布局

生成器以 AIDL package 为目录边界。

规则：

- `package android.test.demo;`
- 输出到 `<out>/android/test/demo/`

这样可以保持与 AIDL package 的稳定映射，并与 Android 现有源码布局一致。

### 2.2 Go package 名

Go package 名取 AIDL package 最后一个 segment，并做 Go 标识符净化：

- 转为小写
- 非法字符替换为 `_`
- 若与 Go 关键字冲突，则追加 `_aidl`

例如：

- `android.test.demo` -> `package demo`
- `android.os` -> `package os`
- `demo.type` -> `package type_aidl`

### 2.3 文件命名

每个 AIDL 源文件默认生成一个 `<basename>_aidl.go`。

例如：

- `IEcho.aidl` -> `iecho_aidl.go`

多文件 package 的聚合与去重由后续阶段 9 的输出管理负责，但命名规则先在这里冻结。

---

## 3. 基础类型映射

| AIDL 类型 | Go 类型 | 说明 |
| --- | --- | --- |
| `void` | 无返回值 | 方法只返回 `error` 或 `(..., error)` |
| `boolean` | `bool` | 直接映射 |
| `byte` | `int8` | AIDL `byte` 是有符号 8 位整数 |
| `char` | `uint16` | 语义为 UTF-16 code unit，不直接映射为 `rune` |
| `int` | `int32` | 固定宽度 |
| `long` | `int64` | 固定宽度 |
| `float` | `float32` | 固定宽度 |
| `double` | `float64` | 固定宽度 |
| `String` | `string` | 非空字符串 |
| `@nullable String` | `*string` | `nil` 表示 null |
| `T[]` | `[]T` | `nil` 表示 null，`len==0` 表示 empty |
| `List<T>` | `[]T` | 与 `T[]` 共享 Go 表达，IR 中保留来源差异 |
| `T[N]` | `[]T` | 公开 API 使用 slice，长度约束与 non-null 约束由生成代码校验 |
| `IBinder` | `binder.Binder` | 低层 Binder 抽象 |
| interface type `IFoo` | `IFoo` | 生成 typed interface；wire 层自动包装 proxy/stub |
| structured parcelable `Foo` | `Foo` | 生成 struct |
| `@nullable Foo` | `*Foo` | `nil` 表示 null |
| enum `Kind` | `Kind` | 生成具名整数类型 |
| union `Result` | `Result` | 生成 tagged union struct |
| custom parcelable `Foo` | 由 sidecar 指定 | 见 custom parcelable 适配规范 |
| `FileDescriptor` | `binder.FileDescriptor` | 显式所有权语义，避免裸 `int` |
| `ParcelFileDescriptor` | `binder.ParcelFileDescriptor` | 显式关闭语义，避免隐式泄漏 |

### 3.1 nullability 规则

Go backend 统一采用下面的 nullable 策略：

- nullable reference-like 类型用 `nil`
- `String` 用 `*string`
- structured parcelable / union 用 `*T`
- interface type 用 `nil` interface
- collection 用 `nil` slice
- `@nullable(heap=true)` 在 Go API 层仍然表现为 nullable，本质差异只保留在 IR 元数据和 codec 约束中

### 3.2 方法边界规则

方法入参、返回值和 `out/inout` 输出与字段的公开类型映射不是完全相同的。

对于非 nullable 的 structured parcelable / union / custom parcelable：

- 字段和普通返回值仍分别表现为 `T` 或 `*T`
- 但方法输入参数统一使用 `*T`
- 方法返回值和 `out/inout` 输出也统一使用 `*T`
- `nil` 不表示 nullable，而是本地参数错误

对应约束：

- client 侧传入 `nil` 时，生成代码直接返回 `binder.ErrBadParcelable`
- client 侧若读到 non-nullable 的 wire null，也返回 `binder.ErrBadParcelable`
- server 侧若收到 wire null，或业务实现返回了 non-nullable 的 `nil`，也返回 `binder.ErrBadParcelable`

这样做的原因是：

- 避免大对象按值传递造成额外拷贝
- 保持 API 更贴近 Go 的“借用对象输入”风格
- 不把“是否 nullable”与“是否值拷贝”绑定在一起
### 3.3 fixed-size array 规则

`T[N]` 固定映射为 `[]T`，但生成代码会在读写时强制校验：

- 长度必须等于 `N`
- 值不能为 `nil`

原因：

- 避免 Go 数组值语义导致的大对象拷贝
- 让公开 API 更符合 Go 调用习惯
- 仍然保留 AIDL fixed-size array 的协议约束
- 约束位置从“编译期类型”转为“生成代码中的运行时校验”

### 3.4 集合元素是否指针化

集合类型默认不把元素整体推广成指针形态。

规则：

- `List<T>` / `T[]` 默认保持 `[]T`
- `Map<K,V>` 默认保持 `map[K]V`
- 不因为元素 `T` 是 parcelable / union / custom parcelable，就自动改成 `[]*T` 或 `map[K]*T`

原因：

- `[]T` 作为方法参数或返回值时只复制 slice header，顶层开销很小
- 真正主要的成本在元素编解码，而不是 slice / map 头部传递
- `[]*T` / `map[K]*T` 会引入更多堆分配、GC 扫描和更差的缓存局部性
- 对 non-nullable 元素类型，元素指针化还会平白引入额外的非法 `nil` 状态

允许的例外：

- AIDL 元素类型本身就是 nullable，此时 Go 元素类型仍可表现为 `*T`
- 例如 `List<@nullable Foo>` 的目标形态可以是 `[]*Foo`

因此，本 backend 的性能优化方向是：

- 单个大对象方法边界使用 `*T`
- 集合本身仍保持 Go 原生 `[]T` / `map[K]V`
- 不把“对象边界指针化”误推广为“集合元素全部指针化”

---

## 4. 方法签名映射

### 4.1 通用规则

每个 generated method 都显式接收 `context.Context` 作为第一个参数。

同步方法的签名规则：

- AIDL return value 优先映射为第一个返回值
- `out` 参数映射为附加返回值
- `inout` 参数映射为输入参数 + 对应更新后的返回值
- `error` 永远作为最后一个返回值

### 4.2 `in` / `out` / `inout`

| AIDL 方向 | Go 形态 |
| --- | --- |
| `in T arg` | 普通类型为 `arg T`；非 nullable parcelable / union / custom parcelable 为 `arg *T` |
| `out T arg` | 普通类型返回 `arg T`；非 nullable parcelable / union / custom parcelable 返回 `arg *T` |
| `inout T arg` | 输入参数 `arg T` 或 `arg *T` + 返回值 `argOut T` 或 `argOut *T` |

示例：

```aidl
interface IFoo {
  int Call(in int a, out String b, inout Payload c);
}
```

生成的 Go 业务接口形态：

```go
type IFoo interface {
    Call(ctx context.Context, a int32, c *Payload) (ret int32, b string, cOut *Payload, err error)
}
```

不采用 C 风格 `*out` 参数。

原因：

- Go 调用方更自然
- 避免生成器把绝大多数签名都变成“传指针等待修改”
- 与 `context.Context` + `error` 的惯常风格更一致

### 4.3 `void` 方法

如果 AIDL 方法返回 `void`：

- 没有 `out/inout` 时，Go 签名为 `Foo(ctx context.Context, ...) error`
- 有 `out/inout` 时，Go 签名为 `Foo(ctx context.Context, ...) (..., error)`

### 4.4 `oneway` 方法

`oneway` 方法保持普通 Go 方法形态，但只返回 `error`：

```go
type IEvents interface {
    Notify(ctx context.Context, event Event) error
}
```

说明：

- client 侧 `error` 只代表本地编码/发送失败
- server 实现如果返回 `error`，该错误不会通过 Binder reply 回给远端，只能用于本地日志或统计

---

## 5. interface / proxy / stub 生成规则

### 5.1 业务接口

每个 AIDL interface 生成一个同名 Go interface，作为业务契约：

```go
type IEcho interface {
    Echo(ctx context.Context, msg string) (string, error)
}
```

### 5.2 client 侧

生成 typed client 构造器：

```go
func NewIEchoClient(target binder.Binder) IEcho
```

约束：

- client helper 返回业务 interface，而不是要求调用方直接操作低层 transaction code
- 具体 proxy struct 可以导出，也可以保持未导出；对外契约固定为返回 typed interface

### 5.3 server 侧

生成 typed handler 构造器：

```go
func NewIEchoHandler(impl IEcho) binder.Handler
```

约束：

- server 实现只关心业务方法
- dispatch、descriptor、transaction code、parcel codec 都由生成代码处理

### 5.4 service helper

生成 typed service helper：

```go
func CheckIEchoService(ctx context.Context, sm binder.ServiceManager, name string) (IEcho, error)
func WaitIEchoService(ctx context.Context, sm binder.ServiceManager, name string) (IEcho, error)
func AddIEchoService(ctx context.Context, sm binder.ServiceManager, name string, impl IEcho, opts ...binder.AddServiceOption) error
```

这样 generated code 只依赖 `binder.ServiceManager`，不直接绑定顶层 `libbinder.Conn`。

---

## 6. 用户定义类型生成规则

### 6.1 structured parcelable

structured parcelable 生成 Go struct。

示例：

```aidl
parcelable Payload {
  int id;
  @nullable String name;
}
```

对应：

```go
type Payload struct {
    ID   int32
    Name *string
}
```

字段规则：

- 字段名转成导出 Go 名称
- 原始 AIDL 名保留在生成元数据中，供 codec 与兼容层使用
- codec 方法由生成器统一补齐

### 6.2 enum

enum 生成具名整数类型，默认 backing type 为 `int32`，若带 `@Backing(type=...)` 则改为对应类型。

示例：

```go
type Kind int32

const (
    KindOne Kind = 1
    KindTwo Kind = 2
)
```

### 6.3 union

union 生成 tagged union struct，不使用 `interface{}`。

目标形态：

```go
type ResultTag int32

const (
    ResultTagCode ResultTag = 1
    ResultTagText ResultTag = 2
)

type Result struct {
    Tag  ResultTag
    Code int32
    Text string
}
```

未被 `Tag` 选中的字段在语义上视为无效。

### 6.4 nested type

nested type 仍生成到同一个 Go package 中，但名称必须带外层前缀，避免冲突。

示例：

- `IEcho.Payload` -> `IEchoPayload`
- `IEcho.Result` -> `IEchoResult`

IR 需要保留原始限定名，公开 Go 名则使用扁平化名称。

---

## 7. stable AIDL 规则

stable AIDL 不直接污染业务接口签名。

冻结规则：

- version/hash 不要求业务实现者手写
- generated stub 自动处理保留 transaction code
- generated proxy 自动缓存远端 version/hash
- generated package 生成稳定常量

目标形态：

```go
const IEchoInterfaceVersion int32 = 3
const IEchoInterfaceHash = "abcdef..."
```

如果远端不支持 stable 保留事务：

- client 侧必须把 `UNKNOWN_TRANSACTION` 当作旧版本信号处理
- generator 产生兼容回退代码，而不是直接把错误抛给业务层

---

## 8. 非目标

下面这些不属于 Go backend 映射规范本身：

- Soong `aidl_interface` 构建规则复刻
- Java/CPP backend 的 API 形式复刻
- 通过 goroutine-local 模拟 Binder 线程 TLS 语义
- 使用裸 transaction code 作为 generated API 的主入口

---

## 9. 本版本结论

`0.0.3` 之后，AIDL 到 Go backend 的公开映射已经冻结：

- 基础类型怎么映射
- `in/out/inout` 怎么出现在 Go 签名里
- proxy / stub / service helper 长什么样
- structured/custom/stable 的边界如何处理

后续实现可以继续补 runtime 和 generator，但不应再反复改这套公开形态。
