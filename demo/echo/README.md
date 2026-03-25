# Echo Demo

这个目录给出一个最小的 Binder `server/client` 通信例子：

- `server/`
  - 打开 Binder 连接
  - 通过 `ServiceManager.AddService` 注册 `libbinder.go.demo.echo`
  - 接收一个字符串并返回 `echo:<msg>`
- `client/`
  - 通过 `ServiceManager.WaitService` 查找同名服务
  - 调用 transaction code `1`
  - 打印返回值

## 目录

```text
demo/echo/
  README.md
  server/main.go
  client/main.go
```

## 协议

- service name: `libbinder.go.demo.echo`
- descriptor: `libbinder.go.demo.IEcho`
- transaction code: `1`

请求体和返回体都只包含一个 `string`。

## 本地构建

```bash
go build ./demo/echo/server
go build ./demo/echo/client
```

## Android arm64 构建

```bash
GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -o ./demo/echo/server/echo-server ./demo/echo/server
GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -o ./demo/echo/client/echo-client ./demo/echo/client
```

## Android 设备运行示例

先推送：

```bash
adb push ./demo/echo/server/echo-server /data/local/tmp/echo-server
adb push ./demo/echo/client/echo-client /data/local/tmp/echo-client
adb shell chmod 755 /data/local/tmp/echo-server /data/local/tmp/echo-client
```

再启动 server：

```bash
adb shell /data/local/tmp/echo-server
```

另一个终端运行 client：

```bash
adb shell '/data/local/tmp/echo-client hello'
```

预期输出类似：

```text
resolved descriptor: libbinder.go.demo.IEcho
request: hello
reply:   echo:hello
```

## 重要限制

这个 demo 依赖 `ServiceManager.AddService`。

在 stock Android emulator 或普通应用域里，`addService` 往往会被 SELinux / service policy 拒绝，所以 `server` 可能直接报错：

```text
addService denied by system policy: binder: remote exception -1: ...
```

因此更适合在下面这些环境里运行：

- userdebug / eng 设备
- 允许 `addService` 的系统域
- 已 root 且策略允许的测试环境

如果只是想看库的实际用法，这个 demo 展示的就是当前推荐的最小调用路径：

1. `Open`
2. `AddService` 或 `WaitService`
3. `Transact`
4. `Close`
