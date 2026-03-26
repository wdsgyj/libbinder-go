# Go 用户态 libbinder 实现计划与步骤

## 1. 目标

本计划面向以下目标：

- 保留现有 Linux/Android kernel Binder driver
- 使用 Go 实现用户态 `libbinder` 对应层
- 先完成最小可用版本，再逐步扩展高级能力

本计划不包括：

- 使用 Go 重写内核 Binder driver
- 首版直接对齐全部 AOSP `libbinder` 功能

## 1.1 当前状态

`0.0.6` 的当前结论：

- 阶段 1 到阶段 11 已全部完成
- 对应代码、单元测试、Android aarch64 模拟器集成测试都已落地
- 当前不再存在路线图中的未完成阶段

`0.0.7` 在此基础上继续补了路线图之外的关键差距收口：

- `ServiceManager` 高级治理能力
- stability 强制校验与分区语义
- RPC 传输辅助与生命周期语义

---

## 2. 总体实施策略

实施顺序建议遵循下面的原则：

1. 先冻结边界，再写代码
2. 先打通 kernel Binder 主链路，再做增强能力
3. 先稳定公开 API 和 internal 分层，再做复杂优化
4. 每一步都要有代码、测试、简短设计说明

建议按 11 个阶段推进。

---

## 3. 阶段 1：范围冻结与 MVP 定义

### 目标

先明确首版到底做什么、不做什么，避免边做边扩。

### 包含内容

1. 明确只做“基于现有 kernel Binder driver 的 Go 用户态实现”
2. 明确 MVP 支持的能力
3. 明确非目标能力
4. 明确验证目标和运行环境

### 建议纳入 MVP 的能力

- `Parcel` 基础读写
- 同步 `Transact`
- oneway `Transact`
- 本地服务注册
- `ServiceManager` 查询
- `ServiceManager` 注册
- 基础 death notification

### 首版暂不做

- Binder RPC 后端
- lazy service
- 稳定性约束
- 录制回放
- 完整缓存策略
- Trusty / TLS / host 特殊适配

### 输出物

- 一份冻结的 MVP 清单
- 一份非目标清单
- 一份环境说明

### 完成标准

- 所有人对首版边界一致
- 后续 API 设计和实现都不再超出这份边界

---

## 4. 阶段 2：公开 API 设计定稿

### 目标

确定用户可见 API，避免 internal 实现反向污染公开接口。

### 包含内容

1. 设计 `binder` 包公开类型
2. 确定错误模型
3. 确定 `context.Context` 的使用方式
4. 确定生命周期 API
5. 确定服务端 handler 接口

### 建议定稿的核心类型

- `Binder`
- `Handler`
- `Parcel`
- `ServiceManager`
- `Subscription`
- `Flags`

### 建议定稿的核心方法

- `Transact(ctx, code, data, flags)`
- `Descriptor(ctx)`
- `CheckService(ctx, name)`
- `WaitService(ctx, name)`
- `AddService(ctx, name, handler, ...)`
- `WatchDeath(ctx)` 或等价订阅接口

### 输出物

- API 草案文档
- package 结构草案
- 错误类型定义草案

### 完成标准

- 不看 internal 实现，使用者也能理解 API
- 公开 API 不直接暴露 driver 细节

---

## 5. 阶段 3：internal 分层与代码骨架搭建

### 目标

先把代码边界和依赖方向定住，再填实现。

### 包含内容

1. 建立 package 目录结构
2. 建立公开层与 internal 层的依赖方向
3. 先放空实现和接口定义
4. 建立 runtime core 与 backend 的分层

### 推荐目录

```text
binder/
internal/runtime/
internal/kernel/
internal/protocol/
```

### 输出物

- 可编译的项目骨架
- 空实现或 stub 实现
- package 依赖说明

### 完成标准

- 项目可以编译
- 公开层不直接依赖 kernel driver 细节
- backend 可以替换，不影响上层接口

---

## 6. 阶段 4：kernel backend 启动层实现

### 目标

建立与现有 kernel Binder driver 通信所需的底座。

### 包含内容

1. 打开 `/dev/binder`
2. 初始化和释放 `mmap`
3. 基础 `ioctl` 桥接
4. 进程级状态对象
5. Binder worker 管理器
6. 绑定 OS 线程的 worker 生命周期

### 关键模块

- `driverManager`
- `processState`
- `workerManager`
- `looperWorker`
- `clientWorker`

### 输出物

- `internal/kernel` 最小启动代码
- 基础 runtime 初始化流程

### 完成标准

- runtime 可以初始化和关闭
- 能创建绑定 OS 线程的 worker
- driver fd、mmap、worker 生命周期正确受控

---

## 7. 阶段 5：Parcel 编解码 MVP 实现

### 目标

实现最小可用的 `Parcel`，支撑真实事务交互。

### 包含内容

1. 标量类型读写
2. 字符串读写
3. `[]byte` 读写
4. Binder 对象编码
5. FD 对象编码
6. object table 管理
7. kernel Binder 结构布局映射
8. reply/status 基础处理

### 建议优先支持的类型

- `int32/int64`
- `uint32/uint64`
- `bool`
- `string`
- `[]byte`
- `Binder`
- file descriptor

### 输出物

- `Parcel` MVP
- codec 单元测试

### 完成标准

- 编解码结果可用于真实 Binder 事务
- 常见类型 round-trip 测试通过

---

## 8. 阶段 6：客户端事务主链路实现

### 目标

打通远端 Binder 调用路径。

### 包含内容

1. remote binder 表示
2. handle 管理
3. `Transact` 路由到 client worker
4. `BINDER_WRITE_READ` 交互
5. `waitForResponse`
6. 同步 reply 处理
7. oneway 路径处理
8. 基础错误映射

### 内部时序

```text
caller goroutine
  -> router
  -> client worker (locked thread)
  -> encode transaction
  -> ioctl
  -> waitForResponse
  -> decode reply/error
  -> return to caller
```

### 输出物

- 可调用远端 Binder 的客户端实现
- 主链路测试

### 完成标准

- 能对真实远端服务发起事务
- 同步调用和 oneway 都能工作

---

## 9. 阶段 7：服务端 looper 与本地节点实现

### 目标

实现本地服务注册和事务处理能力。

### 包含内容

1. 本地 binder 节点表示
2. looper worker 注册进 Binder 线程池
3. 接收 `BR_TRANSACTION`
4. 解码请求 `Parcel`
5. 构造请求上下文
6. 调用本地 `Handler`
7. 编码 reply
8. 回写 driver

### 关键注意点

- 服务端 worker 必须绑定 OS 线程
- calling identity 等上下文应在 worker 内部维护
- handler 默认视为可并发执行，特殊串行需求单独封装

### 输出物

- 本地服务注册和分发实现
- 服务端事务测试

### 完成标准

- Go 服务可被远端进程调用
- reply/error 返回正确

---

## 10. 阶段 8：ServiceManager 集成

### 目标

让用户可以通过标准 Binder 方式查找和注册服务。

### 包含内容

1. context object 获取
2. `CheckService`
3. `WaitService`
4. `AddService`
5. 名称与 Binder 的 facade 封装
6. 基础取消和超时控制

### 输出物

- `ServiceManager` 公开 API
- 服务注册/查询闭环测试

### 完成标准

- 可以查找现有系统服务
- 可以注册并查找到本地 Go 服务

---

## 11. 阶段 9：生命周期与死亡通知实现

### 目标

补齐远端资源生命周期管理和死亡通知。

### 包含内容

1. 远端引用管理
2. 显式 `Release/Close`
3. death subscription
4. `BR_DEAD_BINDER` 处理
5. 订阅取消
6. finalizer 兜底策略

### 建议原则

- 正常逻辑依赖显式释放
- finalizer 仅作为防泄漏兜底
- 订阅统一返回 `Subscription`

### 输出物

- 基础生命周期实现
- death notification 测试

### 完成标准

- 远端服务退出时订阅者能收到通知
- 资源不会长期悬挂

---

## 12. 阶段 10：测试与验证体系

### 目标

建立稳定的自动化验证手段，不靠手工实验。

### 包含内容

1. `Parcel` 单元测试
2. runtime 单元测试
3. mock 测试
4. Android 环境集成测试
5. 服务注册/查询测试
6. 同步/oneway 测试
7. death notification 测试
8. 并发与线程绑定测试

### 建议测试层次

- 快速单元测试
- 需要真实 Binder driver 的集成测试
- 端到端验证

### 输出物

- 测试矩阵
- CI 可运行的测试集合

### 完成标准

- 主链路能力都有自动化覆盖
- 回归时能快速定位问题所在层

---

## 13. 阶段 11：第二阶段增强能力

### 目标

在 MVP 稳定后逐步补齐高级特性。

### 包含内容

1. stability 标签
2. lazy service
3. client cache / addService cache
4. record/replay
5. Binder RPC 后端
6. 调试工具
7. 性能优化

### 建议顺序

1. 先补稳定性和调试能力
2. 再补 ServiceManager 增强能力
3. 最后再做 RPC 后端

### 输出物

- 第二阶段路线图
- 按特性拆分的设计文档

### 完成标准

- 高级能力在不破坏 MVP API 的前提下逐步并入
- `0.0.6` 已完成：
  - stability 标签
  - lazy service
  - client cache / addService cache
  - record/replay
  - Binder RPC backend
  - 调试快照
  - RPC frame pool 与 descriptor/service cache 优化

---

## 14. 推荐实施顺序

建议按照以下顺序推进：

1. 阶段 1-3
   冻结边界、定 API、搭骨架
2. 阶段 4-7
   打通 kernel Binder 主链路
3. 阶段 8-10
   补齐可用性与自动化验证
4. 阶段 11
   做增强能力

简单说就是：

`先定边界 -> 再打通主链路 -> 再补齐测试 -> 最后扩功能`

---

## 15. 每一步都应交付的内容

每个阶段建议至少交付三样东西：

1. 代码
2. 测试
3. 简短设计说明

推荐格式：

- `code`: 实现或骨架
- `test`: 单元/集成验证
- `note`: 本阶段做了什么、没做什么、下一步依赖什么

这样即使中途调整，也不会丢失上下文。

---

## 16. MVP 完成后的判定标准

可以把“MVP 完成”定义成下面这些条件同时满足：

1. 能打开并管理 `/dev/binder`
2. 能发起同步 Binder 调用
3. 能处理 oneway 调用
4. 能注册本地 Go 服务
5. 能通过 `ServiceManager` 查找和注册服务
6. 能收到基础 death notification
7. 有自动化测试覆盖主链路

只要这些条件成立，就说明 Go 用户态 Binder runtime 已经具备“最小可用”价值。
