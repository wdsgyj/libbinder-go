# libbinder-go custom parcelable 适配规范

## 1. 目标

这份文档只解决一种类型：

- `parcelable Foo;`

这类声明在 AIDL 中只给出名字，不给出字段布局。

因此 Go backend 不能像 structured parcelable 一样直接自动生成 struct codec，必须通过额外适配把 AIDL 名称映射到 Go 类型与编解码逻辑。

---

## 2. 设计原则

custom parcelable 的适配必须满足下面几点：

1. 不依赖反射猜测字段布局
2. 不把 Go backend 绑定到某个特定业务包
3. 让生成器在编译时就知道导入路径、Go 类型和 codec 入口
4. 对缺失映射给出明确诊断，而不是运行时才报错

结论：

- `v1` 采用 sidecar 配置文件作为唯一真相源
- 生成器从 sidecar 读取 custom parcelable 映射
- codec 逻辑由业务包提供普通 Go 函数

---

## 3. sidecar 文件

### 3.1 入口

生成器增加一个类型映射参数：

```text
aidlgen -types ./aidl.types.json ...
```

### 3.2 文件格式

`v1` 使用 JSON：

```json
{
  "version": 1,
  "parcelables": [
    {
      "aidl_name": "android.hardware.common.NativeHandle",
      "go_package": "github.com/example/nativehandleaidl",
      "go_type": "NativeHandle",
      "write_func": "WriteNativeHandleToParcel",
      "read_func": "ReadNativeHandleFromParcel",
      "nullable": true
    }
  ]
}
```

字段含义：

- `version`
  - sidecar 格式版本
- `aidl_name`
  - fully-qualified AIDL 类型名
- `go_package`
  - 生成代码需要导入的 Go 包
- `go_type`
  - 非空语义下的 Go 类型名
- `write_func`
  - 写入非空值的函数名
- `read_func`
  - 读取非空值的函数名
- `nullable`
  - 是否允许该 custom parcelable 在 Go API 中出现 nullable 形式

---

## 4. codec 函数签名

sidecar 指向的 codec 函数必须满足下面的签名。

写函数：

```go
func WriteNativeHandleToParcel(p *binder.Parcel, v NativeHandle) error
```

读函数：

```go
func ReadNativeHandleFromParcel(p *binder.Parcel) (NativeHandle, error)
```

约束：

- 函数只处理“非空值”的 codec
- nullable presence marker 由生成代码统一处理
- codec 函数不负责 service helper、Binder object 注册或稳定性元数据

这样设计的原因：

- 生成器可以统一处理 nullability
- codec 只负责具体类型本身的 wire layout
- 不需要强迫业务类型实现某个固定 interface

---

## 5. nullable 规则

如果 AIDL 中使用 `@nullable Foo`：

- sidecar 中必须显式标记 `"nullable": true`
- generated Go API 使用 `*Foo`
- generated codec 在外层写 presence marker
- 非空时再调用 `write_func` / `read_func`

注意：

- `go_type` 必须是“非空语义类型”
- 如果某个类型天然是指针风格，建议封装成具名 wrapper type 后再暴露给生成器
- `v1` 不支持把 `go_type` 直接声明成多层可空指针语义

---

## 6. 生成代码的使用方式

对 custom parcelable `Foo`，生成器的目标代码形态如下：

```go
import custom "github.com/example/nativehandleaidl"

func writeFoo(p *binder.Parcel, v custom.NativeHandle) error {
    return custom.WriteNativeHandleToParcel(p, v)
}

func readFoo(p *binder.Parcel) (custom.NativeHandle, error) {
    return custom.ReadNativeHandleFromParcel(p)
}
```

对 nullable custom parcelable：

```go
func writeNullableFoo(p *binder.Parcel, v *custom.NativeHandle) error {
    if v == nil {
        return p.WriteInt32(0)
    }
    if err := p.WriteInt32(1); err != nil {
        return err
    }
    return custom.WriteNativeHandleToParcel(p, *v)
}
```

presence marker 的具体编码细节由 runtime/codegen 后续统一实现，但接口边界先在这里冻结。

---

## 7. 诊断规则

下面这些情况必须在生成阶段直接报错：

1. `parcelable Foo;` 没有对应 sidecar 条目
2. sidecar 中同一个 `aidl_name` 出现重复定义
3. `go_package` 或 `go_type` 为空
4. `write_func` / `read_func` 缺失
5. AIDL 使用了 nullable，但 sidecar 没有声明 `nullable: true`
6. structured parcelable 被错误地放进 custom parcelable sidecar

诊断信息必须包含：

- AIDL 源文件
- 类型名
- sidecar 文件路径
- 失败原因

---

## 8. 与 built-in 特殊类型的边界

下面这些不走 custom parcelable sidecar：

- `String`
- `IBinder`
- interface type
- `FileDescriptor`
- `ParcelFileDescriptor`
- structured parcelable
- enum
- union

这些类型要么是 AIDL built-in，要么有单独的 runtime/codegen 语义。

---

## 9. 未来扩展

`v1` 先只支持 sidecar + codec 函数。

后续可以扩展：

- YAML sidecar
- 代码注册表
- 预置 AOSP 常见 custom parcelable 映射表
- 自动校验 codec 符号是否存在

但扩展不能破坏 `v1` 的基本约束：

- fully-qualified AIDL name -> Go type
- write/read 函数显式指定
- 缺失映射在生成期失败

---

## 10. 结论

custom parcelable 的核心不是“自动生成 struct”，而是“把 opaque AIDL 类型稳定接到 Go 类型系统里”。

`0.0.3` 之后这件事的结论已经明确：

- 用 sidecar 做类型映射
- 用显式 codec 函数做非空值编解码
- 用生成器统一处理 nullability 和诊断

这样后续阶段 6 可以直接实现，不需要再回头重新设计接入方式。
