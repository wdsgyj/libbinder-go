# Android aarch64 模拟器测试说明

## 1. 目的

这套脚本用于把 `libbinder-go` 的测试环境固定到 Android aarch64 模拟器，而不是宿主机 Darwin/Linux。

当前目标有两层：

- 在宿主机上把整个模块交叉编译到 `GOOS=android` `GOARCH=arm64`
- 在 Android 模拟器里运行 Go 测试二进制

这样可以尽早暴露：

- Android build tag 问题
- Android libc / linker 兼容问题
- `/dev/binder` 基础打开与 `mmap` 能力问题

---

## 2. 相关文件

- `scripts/android-emulator-test.sh`
- `scripts/lib/android-emulator-common.sh`
- `internal/kernel/backend_android_test.go`
- `internal/kernel/driver_transact_android_test.go`
- `internal/kernel/transact.go`
- `internal/kernel/uapi_64.go`
- `internal/kernel/driver_uapi_64.go`
- `internal/runtime/transact.go`
- `internal/runtime/runtime_android_test.go`

---

## 3. 当前测试链路

脚本执行顺序如下：

1. 检测 Android SDK / emulator / adb / avdmanager / sdkmanager
2. 确保目标 Android system image 已安装
3. 确保目标 AVD 已创建
4. 启动 Android aarch64 模拟器并等待开机完成
5. 使用 `GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build ./...` 做一次整模块 Android 编译检查
6. 找出带测试文件的 Go package
7. 为每个 package 构建 Android 测试二进制
8. 通过 `adb push` 下发到模拟器
9. 在模拟器中执行测试二进制

---

## 4. 默认配置

默认值如下：

- `ANDROID_API_LEVEL=35`
- `ANDROID_IMAGE_TAG=google_apis`
- `ANDROID_ABI=arm64-v8a`
- `ANDROID_DEVICE_PROFILE=medium_phone`
- `ANDROID_AVD_NAME=libbinder-go-api35-arm64-v8a`
- `ANDROID_EMULATOR_PORT=5560`
- `REMOTE_TEST_DIR=/data/local/tmp/libbinder-go-tests`

这些值都可以通过环境变量覆盖。

---

## 5. 使用方式

### 5.1 跑全部测试

```bash
./scripts/android-emulator-test.sh
```

### 5.2 只跑指定 package

```bash
./scripts/android-emulator-test.sh ./internal/kernel
```

### 5.3 传递测试二进制参数

```bash
./scripts/android-emulator-test.sh ./internal/kernel -- -test.v -test.run TestDriverManagerOpenCloseOnAndroid
```

### 5.4 复用已启动的 emulator

```bash
ANDROID_SERIAL=emulator-5560 ./scripts/android-emulator-test.sh
```

### 5.5 保留 emulator 不自动关闭

```bash
ANDROID_KEEP_EMULATOR=1 ./scripts/android-emulator-test.sh
```

---

## 6. 当前 Android 测试覆盖

目前已经落地两类 Android 专用测试：

- `internal/kernel/backend_android_test.go`
- `internal/kernel/driver_transact_android_test.go`
- `internal/runtime/runtime_android_test.go`

### 6.1 backend 生命周期

1. `DriverManager` 能在 Android 环境打开 `/dev/binder`
2. `mmap` 已建立
3. `Backend.Start()` 能拉起 thread-bound worker
4. `Backend.Close()` 能正确回收资源

### 6.2 driver ioctl 与最小真实事务

1. `BINDER_VERSION` 返回协议版本
2. `BINDER_WRITE_READ` 空调用可执行
3. `BC_ENTER_LOOPER` 可执行
4. 能对 handle `0` 发起 `PING_TRANSACTION`
5. 能读到 `BR_TRANSACTION_COMPLETE` 和 `BR_REPLY`
6. 能正确发送 `BC_FREE_BUFFER`

这一步的意义是先确认：

- Android 上不是落到 unsupported stub
- Go runtime + Android linker + Binder driver 的最基础交互已经成立
- 最小真实 Binder request/reply 主链路已经通了

### 6.3 runtime 级同步 transact

1. `Runtime.Start()` 能完成 driver + worker 初始化
2. `Runtime.TransactHandle()` 会通过 client worker 在锁定的 OS 线程上执行事务
3. runtime 级 `PING_TRANSACTION` 能返回空 reply `Parcel`

这一步的意义是先确认：

- `Runtime -> Kernel Backend -> ClientWorker -> Driver` 这条调用链已经成立
- 事务执行没有退回到“普通 goroutine 直接 ioctl”这种错误模型

---

## 7. 当前边界

这套脚本已经能验证 Android 运行环境、backend 生命周期、driver 级最小真实 request/reply，以及 runtime 级同步 transact 主链路，但还没有覆盖完整高层 Binder 事务链路。

当前未覆盖的能力包括：

- 同步事务 / oneway 事务
- `ServiceManager` 交互
- death notification

原因很简单：这些功能在当前代码里还没有完全实现，测试脚本不能替代尚未落地的 runtime 逻辑。

---

## 8. 后续建议

当下面这些能力实现后，应该直接把 Android 模拟器脚本继续扩成集成测试入口：

1. client transact framing
2. 把当前 driver 级 ping 测试提升成 runtime 级 transact API 测试
3. reply/status 完整解码
4. local service registration
5. `ServiceManager` 查询/注册
6. death notification

到那时，这套脚本就可以从“环境 smoke test”升级为真正的 Binder 集成测试入口。
