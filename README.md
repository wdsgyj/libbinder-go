# libbinder-go

`libbinder-go` 是一个用 Go 重写 Android 用户态 `libbinder` 的项目。

它的目标不是重写 Linux kernel Binder driver，而是基于现有 `/dev/binder` 和 Binder RPC 语义，提供一套更符合 Go 风格的 runtime、AIDL 代码生成器和配套工具链。

## 项目内容

当前仓库主要包含这些部分：

- kernel Binder runtime
  - 打开 `/dev/binder`
  - 发起同步 / oneway 事务
  - 本地 handler 注册与分发
  - death notification
  - `ServiceManager` 查询、注册和治理能力
- Binder RPC runtime
  - `DialRPC*` / `ServeRPC*`
  - `tcp` / `unix` / `tls` 传输辅助
  - 会话内 Binder object、callback、death 语义
- AIDL -> Go 工具链
  - parser / resolve / IR / Go model / codegen
  - `cmd/aidlgen` 命令行工具
  - custom parcelable / stable AIDL / 多文件输出支持
- AOSP `cmd` 的 Go 实现
  - `cmd/cmd`
  - 支持 `-l`、`-w`、shell command transact、`IShellCallback`、`IResultReceiver`
- 示例与文档
  - `demo/echo` 提供最小 server/client 通信例子
  - `doc/` 下保存分析、路线图、实现计划和架构文档
  - `aosp-src/` 下保留上游 AOSP 参考源码

## 目标

项目的长期目标是：

- 用 Go 风格 API 提供 Android Binder 用户态能力，而不是机械复刻 C++ API
- 保留 Binder 必需的 thread-bound 语义，不用 goroutine local 假装成 thread local
- 提供可落地的 AIDL 到 Go client/server 样板代码生成能力
- 在宿主机和 Android aarch64 模拟器/设备上都具备自动化验证

不在目标内的内容：

- 用 Go 重写 kernel Binder driver
- 规避 Android 系统服务注册权限、SELinux 或分区策略

## 当前目录

```text
.
├── aosp-src/         # AOSP 参考源码
├── binder/           # 对外公开 API 类型
├── cmd/
│   ├── aidlgen/      # AIDL -> Go 代码生成器
│   └── cmd/          # AOSP cmd 的 Go 实现
├── demo/
│   └── echo/         # 最小 Binder server/client 示例
├── doc/              # 设计、分析、路线图、架构文档
├── internal/         # runtime、kernel、protocol、aidl 内部实现
└── scripts/          # 测试与辅助脚本
```

## 环境要求

- Go `1.22+`
- 如果要使用 kernel Binder backend，需要运行环境具备 `/dev/binder`
- 如果要在 Android 设备上运行 `cmd`、demo server 或 `AddService` 相关代码，需要系统策略允许对应 Binder 行为

注意：

- 在 macOS 或普通 Linux 主机上直接运行 `go run ./cmd/cmd -l`，通常拿不到默认 `ServiceManager`，因为本机没有 Android kernel Binder 环境
- `cmd` 更适合交叉编译为 Android 二进制后通过 `adb shell` 运行

## 使用方式

### 1. 直接使用 Binder runtime

最小 client 侧用法：

```go
package main

import (
	"context"
	"log"

	"github.com/wdsgyj/libbinder-go"
)

func main() {
	conn, err := libbinder.Open(libbinder.Config{})
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	service, err := conn.ServiceManager().WaitService(context.Background(), "SurfaceFlinger")
	if err != nil {
		log.Fatal(err)
	}
	defer service.Close()

	desc, err := service.Descriptor(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	log.Println("descriptor:", desc)
}
```

如果要看更完整的 server/client 交互，直接参考 `demo/echo`。

### 2. 从 AIDL 生成 Go 代码

查看 AIDL 摘要：

```bash
go run ./cmd/aidlgen path/to/IFoo.aidl
```

输出 Go 代码到目录：

```bash
go run ./cmd/aidlgen -format go -out ./gen path/to/IFoo.aidl
```

如果有 custom parcelable 或 stable interface 映射：

```bash
go run ./cmd/aidlgen -format go -types ./types.json -out ./gen path/to/IFoo.aidl
```

### 3. 构建并运行 `cmd`

为 Android arm64 构建：

```bash
GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -o /tmp/libbinder-go-cmd ./cmd/cmd
```

推送到设备并列出服务：

```bash
adb push /tmp/libbinder-go-cmd /data/local/tmp/libbinder-go-cmd
adb shell chmod 755 /data/local/tmp/libbinder-go-cmd
adb shell /data/local/tmp/libbinder-go-cmd -l
```

等待某个 service 并执行 shell command：

```bash
adb shell '/data/local/tmp/libbinder-go-cmd -w activity services'
```

说明：

- `cmd` 依赖目标设备的 Binder / ServiceManager 环境
- 不建议在非 Android 主机上直接 `go run ./cmd/cmd -l`

### 4. 运行 echo demo

构建：

```bash
GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -o /tmp/echo-server ./demo/echo/server
GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -o /tmp/echo-client ./demo/echo/client
```

推送并运行：

```bash
adb push /tmp/echo-server /data/local/tmp/echo-server
adb push /tmp/echo-client /data/local/tmp/echo-client
adb shell chmod 755 /data/local/tmp/echo-server /data/local/tmp/echo-client
adb shell /data/local/tmp/echo-server
adb shell '/data/local/tmp/echo-client hello'
```

如果设备策略不允许 `addService`，server 可能会因为 SELinux / service policy 被拒绝。这是系统环境限制，不是项目编码错误。

## 测试

宿主机：

```bash
go test ./...
```

Android aarch64 模拟器：

```bash
ANDROID_AVD_NAME=Medium_Phone ANDROID_SKIP_SDK_INSTALL=1 ANDROID_HEADLESS=1 ANDROID_WIPE_DATA=0 ./scripts/android-emulator-test.sh ./... -- -test.v
```

## 相关文档

- [CHANGELOG.md](./CHANGELOG.md)
- [doc/libbinder-go-implementation-roadmap.md](./doc/libbinder-go-implementation-roadmap.md)
- [doc/libbinder-go-aidl-full-plan.md](./doc/libbinder-go-aidl-full-plan.md)
- [doc/libbinder-go-runtime-internal-architecture.md](./doc/libbinder-go-runtime-internal-architecture.md)
- [doc/cmd-tool-analysis.md](./doc/cmd-tool-analysis.md)
- [demo/echo/README.md](./demo/echo/README.md)
