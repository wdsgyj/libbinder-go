# aidlgen 使用文档

## 1. 目标

`aidlgen` 是本仓库的 AIDL 解析与 Go 代码生成工具。

它支持：

- 解析 AIDL AST
- 输出摘要级 model
- 将 AIDL 生成为 Go 的 interface / client / handler / service helper / parcelable codec
- 通过 sidecar JSON 补充 custom parcelable 与 stable interface 元数据

源码入口见 [main.go](../cmd/aidlgen/main.go)。

---

## 2. 快速开始

直接运行：

```bash
go run ./cmd/aidlgen -format summary path/to/IFoo.aidl
```

或者先编译：

```bash
go build -o ./bin/aidlgen ./cmd/aidlgen
./bin/aidlgen -format go -out ./gen path/to/IFoo.aidl
```

---

## 3. 命令行参数

`aidlgen` 当前支持这些参数：

```text
-format         输出格式：summary、ast、model、go
-out            当 -format go 时的输出目录
-types          sidecar JSON，补充 custom parcelable / stable interface 映射
-go-import-root 生成代码的 Go import 根路径，跨 package 生成时必需
-roots-only     只输出命令行指定的 root AIDL 文件，不输出自动加载的依赖文件
```

注意：

- `-format go` 且不传 `-out` 时，只允许单文件单输出。
- 如果输入 AIDL 跨 package 引用了其他生成结果，必须传 `-go-import-root`，否则会报错。
- `-roots-only` 只影响“输出哪些文件”，不影响依赖解析。依赖文件仍会被读取用于类型解析。

---

## 4. 输出模式

### 4.1 `-format summary`

输出简化后的 IR 摘要，适合快速看声明结构。

```bash
go run ./cmd/aidlgen -format summary demo/IFoo.aidl
```

### 4.2 `-format ast`

输出 parser AST，适合调试语法解析。

```bash
go run ./cmd/aidlgen -format ast demo/IFoo.aidl
```

### 4.3 `-format model`

输出 typed Go model，适合调试类型 lowering 和 codegen 前状态。

```bash
go run ./cmd/aidlgen -format model demo/IFoo.aidl
```

### 4.4 `-format go`

生成 Go 代码。

```bash
go run ./cmd/aidlgen -format go -out ./gen demo/IFoo.aidl
```

---

## 5. 输出布局规则

### 5.1 目录布局

如果 AIDL 文件带 package：

```aidl
package android.test.demo;
```

会输出到：

```text
<out>/android/test/demo/
```

如果 AIDL 文件没有 package：

- 生成器会优先用源码所在目录名作为 Go package 名
- 同目录的无 package 多文件会归并到同一个 Go package / 输出目录

这条规则是为了兼容 AOSP binder/tests 这类无 package 语料。

### 5.2 Go package 名

Go package 名默认取 AIDL package 最后一个 segment，并做净化：

- 全部转小写
- 非法字符改为 `_`
- 若与 Go 关键字冲突，追加 `_aidl`

示例：

- `android.test.demo` -> `package demo`
- `android.os` -> `package os`
- `demo.type` -> `package type_aidl`

### 5.3 文件命名

默认按源文件名生成：

- `IFoo.aidl` -> `ifoo_aidl.go`
- `Payload.aidl` -> `payload_aidl.go`

---

## 6. AIDL 到 Go 的类型映射

下表是当前仓库真实实现对应的公开 Go 映射。

| AIDL 类型 | Go 类型 | 说明 |
| --- | --- | --- |
| `void` | 无返回值 | 方法只返回 `error` 或 `(..., error)` |
| `boolean` | `bool` | |
| `byte` | `int8` | AIDL `byte` 是有符号 8 位整数 |
| `char` | `uint16` | UTF-16 code unit，不直接映射成 `rune` |
| `int` | `int32` | 固定宽度 |
| `long` | `int64` | 固定宽度 |
| `float` | `float32` | |
| `double` | `float64` | |
| `String` | `string` | 非 nullable |
| `@nullable String` | `*string` | `nil` 表示 null |
| `T[]` | `[]T` | slice，`nil` 表示 null |
| `T[N]` | `[]T` | 公开 API 使用 slice，生成代码强制校验长度必须等于 `N` |
| `List<T>` | `[]T` | Go 层与动态数组共用 slice |
| `Map<K,V>` | `map[K]V` | typed map |
| `Map` | `map[any]any` | raw map / dynamic map |
| `IBinder` | `binder.Binder` | 低层 Binder 抽象 |
| `interface IFoo` | `IFoo` | 生成 typed interface |
| structured `parcelable Foo` | `Foo` | 生成 struct |
| `@nullable Foo` | `*Foo` | nullable parcelable |
| `enum Kind` | `Kind` | 具名整数类型 |
| `union Result` | `Result` | tagged union struct |
| `FileDescriptor` | `binder.FileDescriptor` | 原始 FD |
| `ParcelFileDescriptor` | `binder.ParcelFileDescriptor` | owned FD 包装 |
| `@nullable ParcelFileDescriptor` | `*binder.ParcelFileDescriptor` | nullable PFD |
| non-structured `parcelable Foo;` | 默认生成 opaque fallback | 或由 `-types` sidecar 自定义 |

### 6.1 nullable 规则

当前实现里，nullable 主要支持这些类别：

- `@nullable String` -> `*string`
- `@nullable Foo` -> `*Foo`
- `@nullable Union` -> `*Union`
- `@nullable ParcelFileDescriptor` -> `*binder.ParcelFileDescriptor`
- interface 类型天然可空
- `T[]` / `List<T>` / `Map<K,V>` 通过 `nil` 表示 null

当前不支持把基础标量直接 nullable：

- `@nullable int`
- `@nullable boolean`
- `@nullable long`

这类会在 lowering 阶段报诊断。

### 6.2 `List<T>` 与 `T[]`

两者在 Go API 上都映射成 `[]T`，但生成器内部仍区分来源，以便保持 wire codec 语义一致。

### 6.3 `Map`

支持两种形式：

```aidl
Map<String, Payload>
Map
```

对应：

```go
map[string]Payload
map[any]any
```

注意：

- `raw Map` 的 Go 侧已支持
- 但 Android Java SDK AIDL 工具本身不接受很多 untyped `Map` 接口定义，所以 Java 侧常需要手写 Binder shim 才能互通

### 6.4 `@utf8InCpp`

`@utf8InCpp String` 在 Go API 上仍然表现为 `string` 或 `*string`，不会引入额外 Go 类型。

### 6.5 enum / union

示例：

```aidl
enum Kind { ONE = 1, TWO = 2 }

union Result {
  int code;
  @nullable String text;
}
```

会生成：

```go
type Kind int32

type Result struct {
    Tag  ResultTag
    Code int32
    Text *string
}
```

### 6.6 non-structured parcelable

对这种声明：

```aidl
parcelable Foo;
```

默认生成器会给出一个可编译的 opaque fallback 类型，用于先打通生成链路。

如果你想要真实编解码语义，应该通过 `-types` 提供自定义 codec 映射。

---

## 7. 方法签名映射

### 7.1 基本规则

生成后的业务接口总是以 `context.Context` 为第一个参数。

规则：

- AIDL return value -> Go 第一个返回值
- `out` 参数 -> 额外返回值
- `inout` 参数 -> 输入参数 + 更新后的输出返回值
- `error` 永远是最后一个返回值

示例：

```aidl
interface IFoo {
  int Call(in int a, out String b, inout Payload c);
}
```

生成后大致为：

```go
type IFoo interface {
    Call(ctx context.Context, a int32, c Payload) (int32, string, Payload, error)
}
```

### 7.2 `oneway`

`oneway` 仍生成普通 Go 方法，但客户端只会拿到本地发送层面的 `error`：

```aidl
oneway void Notify(in String msg);
```

对应：

```go
Notify(ctx context.Context, msg string) error
```

---

## 8. 生成结果包含什么

对一个 interface，生成器通常会生成：

- 业务接口定义
- typed client
- typed handler
- transaction code 常量
- descriptor 常量
- `WriteXxxToParcel` / `ReadXxxFromParcel`
- `CheckXxxService` / `WaitXxxService` / `AddXxxService`

可参考生成结果：

- [ibaselineservice_aidl.go](../tests/aidl/go/shared/generated/com/wdsgyj/libbinder/aidltest/shared/ibaselineservice_aidl.go)
- [ibasicmatrixservice_aidl.go](../tests/aidl/go/shared/generated/com/wdsgyj/libbinder/aidltest/shared/ibasicmatrixservice_aidl.go)

---

## 9. `aidl.types.json`

`-types` 指向一个 sidecar JSON 文件，当前有两个作用：

- custom parcelable 映射
- stable interface 的 version/hash 映射

示例见 [aidl.types.json](../tests/aidl/extra/aidl/aidl.types.json)。

### 9.1 文件格式

```json
{
  "version": 1,
  "interfaces": [
    {
      "aidl_name": "com.example.IFoo",
      "version": 3,
      "hash": "foo-hash-v3"
    }
  ],
  "parcelables": [
    {
      "aidl_name": "com.example.CustomBox",
      "go_package": "example.com/project/customcodec",
      "go_type": "CustomBox",
      "write_func": "WriteCustomBoxToParcel",
      "read_func": "ReadCustomBoxFromParcel",
      "nullable": true
    }
  ]
}
```

### 9.2 custom parcelable 字段含义

| 字段 | 含义 |
| --- | --- |
| `aidl_name` | AIDL 全限定名 |
| `go_package` | 自定义 codec 所在 Go 包 |
| `go_type` | 公开 Go 类型名 |
| `write_func` | 写入 parcel 的函数 |
| `read_func` | 从 parcel 读取的函数 |
| `nullable` | 是否允许 nullable |

### 9.3 stable interface 字段含义

| 字段 | 含义 |
| --- | --- |
| `aidl_name` | AIDL 全限定 interface 名 |
| `version` | interface version，必须大于 0 |
| `hash` | interface hash |

如果 AIDL interface 带稳定性语义，但没有在 `-types` 里提供 version/hash，生成器会失败。

---

## 10. Demo 1：单文件生成

### 10.1 AIDL

`demo/IEcho.aidl`

```aidl
package demo;

parcelable Payload {
  int id;
  @nullable String note;
}

interface IEcho {
  @nullable String Echo(in String msg, out int code, inout Payload payload);
}
```

### 10.2 生成命令

```bash
go run ./cmd/aidlgen -format go -out ./gen demo/IEcho.aidl
```

生成结果：

```text
gen/demo/iecho_aidl.go
```

### 10.3 生成后的接口形态

大致会生成：

```go
type IEcho interface {
    Echo(ctx context.Context, msg string, payload Payload) (*string, int32, Payload, error)
}
```

---

## 11. Demo 2：多文件同包生成

目录：

```text
demo/
  Payload.aidl
  IService.aidl
```

`demo/Payload.aidl`

```aidl
package demo;

parcelable Payload {
  int id;
}
```

`demo/IService.aidl`

```aidl
package demo;

interface IService {
  Payload Echo(in Payload value);
}
```

生成：

```bash
go run ./cmd/aidlgen -format go -out ./gen demo/IService.aidl
```

这里即使只把 `IService.aidl` 作为 root 传入，生成器也会自动加载同包依赖并输出：

```text
gen/demo/iservice_aidl.go
gen/demo/payload_aidl.go
```

如果你只想输出 root 文件本身：

```bash
go run ./cmd/aidlgen -format go -roots-only -out ./gen demo/IService.aidl
```

这样只会输出：

```text
gen/demo/iservice_aidl.go
```

---

## 12. Demo 3：跨 package 生成

如果 AIDL 跨 package 引用其他生成结果，必须指定 `-go-import-root`。

示例：

```text
alpha/Foo.aidl
gamma/ICallback.aidl
beta/IBar.aidl
```

生成：

```bash
go run ./cmd/aidlgen \
  -format go \
  -go-import-root example.com/generated \
  -out ./gen \
  alpha/Foo.aidl \
  gamma/ICallback.aidl \
  beta/IBar.aidl
```

生成代码会自动写出类似：

```go
import (
    alpha "example.com/generated/alpha"
    gamma "example.com/generated/gamma"
)
```

---

## 13. Demo 4：custom parcelable

### 13.1 AIDL

```aidl
package demo;

parcelable CustomBox;

interface ICustomService {
  @nullable CustomBox Echo(in CustomBox value);
}
```

### 13.2 自定义 codec

```go
package customcodec

import "github.com/wdsgyj/libbinder-go/binder"

type CustomBox struct {
    Name string
    ID   int32
}

func WriteCustomBoxToParcel(p *binder.Parcel, v *CustomBox) error {
    if v == nil {
        return p.WriteInt32(0)
    }
    if err := p.WriteInt32(1); err != nil {
        return err
    }
    if err := p.WriteString(v.Name); err != nil {
        return err
    }
    return p.WriteInt32(v.ID)
}

func ReadCustomBoxFromParcel(p *binder.Parcel) (*CustomBox, error) {
    present, err := p.ReadInt32()
    if err != nil {
        return nil, err
    }
    if present == 0 {
        return nil, nil
    }
    name, err := p.ReadString()
    if err != nil {
        return nil, err
    }
    id, err := p.ReadInt32()
    if err != nil {
        return nil, err
    }
    return &CustomBox{Name: name, ID: id}, nil
}
```

### 13.3 sidecar

```json
{
  "version": 1,
  "parcelables": [
    {
      "aidl_name": "demo.CustomBox",
      "go_package": "example.com/project/customcodec",
      "go_type": "CustomBox",
      "write_func": "WriteCustomBoxToParcel",
      "read_func": "ReadCustomBoxFromParcel",
      "nullable": true
    }
  ]
}
```

### 13.4 生成命令

```bash
go run ./cmd/aidlgen \
  -format go \
  -types ./aidl.types.json \
  -out ./gen \
  demo/ICustomService.aidl
```

此时生成代码会直接引用你的 codec 包，而不是生成 opaque fallback。

---

## 14. Demo 5：stable interface

如果 interface 需要带 version/hash：

```json
{
  "version": 1,
  "interfaces": [
    {
      "aidl_name": "demo.IMetadataService",
      "version": 7,
      "hash": "metadata-hash-v7"
    }
  ]
}
```

生成：

```bash
go run ./cmd/aidlgen \
  -format go \
  -types ./aidl.types.json \
  -out ./gen \
  demo/IMetadataService.aidl
```

生成后的 handler / client 会带上 interface version/hash 支持。

---

## 15. Demo 6：生成后如何写 Go server

下面是典型 server 侧用法，风格与测试夹具一致。

```go
package main

import (
    "context"

    libbinder "github.com/wdsgyj/libbinder-go"
    generated "example.com/generated/demo"
)

type echoServer struct{}

func (echoServer) Echo(ctx context.Context, msg string, payload generated.Payload) (*string, int32, generated.Payload, error) {
    reply := "go:" + msg
    payload.ID++
    return &reply, 200, payload, nil
}

func main() {
    conn, err := libbinder.Open(libbinder.Config{
        DriverPath:    "/dev/binder",
        LooperWorkers: 1,
        ClientWorkers: 1,
    })
    if err != nil {
        panic(err)
    }
    defer conn.Close()

    if err := generated.AddIEchoService(context.Background(), conn.ServiceManager(), "demo.echo", echoServer{}); err != nil {
        panic(err)
    }

    select {}
}
```

实际参考：

- [baseline server](../tests/aidl/go/server/baseline/main.go)

---

## 16. Demo 7：生成后如何写 Go client

```go
package main

import (
    "context"
    "fmt"
    "time"

    libbinder "github.com/wdsgyj/libbinder-go"
    generated "example.com/generated/demo"
)

func main() {
    conn, err := libbinder.Open(libbinder.Config{DriverPath: "/dev/binder"})
    if err != nil {
        panic(err)
    }
    defer conn.Close()

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    svc, err := generated.WaitIEchoService(ctx, conn.ServiceManager(), "demo.echo")
    if err != nil {
        panic(err)
    }

    msg := "hello"
    out, code, payload, err := svc.Echo(ctx, msg, generated.Payload{ID: 7})
    if err != nil {
        panic(err)
    }

    fmt.Println(*out, code, payload.ID)
}
```

实际参考：

- [baseline client](../tests/aidl/go/client/baseline/main.go)

---

## 17. 常见问题

### 17.1 为什么不传 `-out` 会失败？

因为 `-format go` 不带 `-out` 时，只允许单输出文件。如果输入导致生成多个文件，会报：

```text
expected single output, got N; use -out for multi-file generation
```

### 17.2 为什么跨 package 生成报错？

因为缺少：

```bash
-go-import-root example.com/generated
```

跨 package 的生成代码需要稳定 import 路径。

### 17.3 为什么 `Map` 在 Java AIDL 侧不好用？

Go 生成器支持 raw `Map -> map[any]any`，但 Android Java SDK AIDL 工具本身对 untyped `Map` 有限制。这个限制来自 Java AIDL 工具，不是 Go 生成器。

### 17.4 为什么 non-structured parcelable 需要 `-types`？

因为：

```aidl
parcelable Foo;
```

本身没有字段布局，生成器无法推导真实编解码逻辑。默认只能给出编译级 fallback；要做真实互通，必须显式提供 codec。

---

## 18. 相关文档

- [libbinder-go AIDL Go Backend 映射规范](./libbinder-go-aidl-go-backend-mapping.md)
- [libbinder-go AIDL 支持矩阵](./libbinder-go-aidl-support-matrix.md)
- [AIDL 全量兼容测试框架](./aidl-full-compat-test-framework.md)
