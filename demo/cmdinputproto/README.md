# cmd input Protocol Demo

这个 demo 不做真实 input 注入。

它只验证 `cmd input` 这条 shell-command 协议链路里的几件事：

- request parcel 的结构
- `source` / `-d DISPLAY_ID` 的前置解析
- help / known command / unknown command 三类分支
- `ResultReceiver` 的逻辑结果回传
- `ShellCallback` 在 `input` 路径中未被使用

## 目录

```text
demo/cmdinputproto/
  README.md
  protocol.go
  protocol_test.go
```

## 运行

```bash
go test -cover ./demo/cmdinputproto
```

这个 demo 是协议模拟器，不依赖 Android 设备，也不要求 `/dev/binder`。

## 设计边界

- 它模拟的是 `cmd input` 的协议和 shell framework 语义
- 它不模拟真实 `MotionEvent` / `KeyEvent` 注入
- 它的目的不是替代系统 `input` service，而是把协议职责边界验证清楚
