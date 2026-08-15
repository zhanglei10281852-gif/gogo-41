# Bug Reproduction

## 包的性质

当前 test_model_fix 保存的是被测模型修复后的结果源码，不是初始含 Bug 源码。要复现原始缺陷，必须检出下面固定的 parent SHA；不要在当前修复结果源码上期待重新出现修复前失败。生成系统使用的可信验证补丁和完整验证日志仅在本地留存，不提交到结果分支。

## 问题现象

工作者持有的任务租约已经过期且超过 heartbeat grace 后，失败上报仍被接受，任务从 leased 变成 retry_wait。请修复过期租约处理：迟到的失败上报必须返回错误并完整保留原租约、状态、历史和错误信息，同时不要影响宽限期内的合法失败上报，并保证全量测试通过。

## 含 Bug 版本

- 仓库：zhanglei10281852-gif/gogo-41
- 仓库地址：https://github.com/zhanglei10281852-gif/gogo-41.git
- parent SHA：ece32bf3d127bb69f274f5a1b47bac34990610b4

## 复现步骤

```bash
git clone -- https://github.com/zhanglei10281852-gif/gogo-41.git bug-repro
cd bug-repro
git checkout --detach ece32bf3d127bb69f274f5a1b47bac34990610b4
go test ./internal/engine -run "^TestFailRejectsLeaseAfterExpiryGrace$" -count=1 -v
```

## 双架构完整错误信息

### linux/amd64

- 容器内复现预期退出码：1
- 容器内复现实际退出码：1

stdout：

```text
$ go test ./internal/engine -run "^TestFailRejectsLeaseAfterExpiryGrace$" -count=1 -v
=== RUN   TestFailRejectsLeaseAfterExpiryGrace
    engine_test.go:87: Fail() = (&{ID:expired-fail Queue:default Type:work Payload:[110 117 108 108] Metadata:map[] Priority:0 State:retry_wait CreatedAt:2023-11-14 22:13:20 +0000 UTC UpdatedAt:2023-11-14 22:13:32.000000001 +0000 UTC AvailableAt:2023-11-14 22:13:37.000000001 +0000 UTC Deadline:<nil> Dependencies:[] MaxAttempts:3 Attempts:1 Backoff:{Kind:exponential BaseSeconds:5 MaxSeconds:3600 JitterPercent:10} RequiredLabels:map[] Resources:{CPU:0 MemoryMB:0 Slots:1} IdempotencyKey: Lease:<nil> Result:[] LastError:0xc00007aac0 History:[{From: To:ready At:2023-11-14 22:13:20 +0000 UTC Reason:enqueued and ready Actor:enqueue} {From:ready To:leased At:2023-11-14 22:13:20 +0000 UTC Reason:claimed Actor:worker} {From:leased To:retry_wait At:2023-11-14 22:13:32.000000001 +0000 UTC Reason:retry scheduled in 5s Actor:worker}]}, <nil>), want nil job and expired-lease error
--- FAIL: TestFailRejectsLeaseAfterExpiryGrace (0.03s)
FAIL
FAIL	QueueForge/internal/engine	0.034s
FAIL

```

stderr：

```text
(empty)
```

### linux/arm64

- 容器内复现预期退出码：1
- 容器内复现实际退出码：1

stdout：

```text
$ go test ./internal/engine -run "^TestFailRejectsLeaseAfterExpiryGrace$" -count=1 -v
=== RUN   TestFailRejectsLeaseAfterExpiryGrace
    engine_test.go:87: Fail() = (&{ID:expired-fail Queue:default Type:work Payload:[110 117 108 108] Metadata:map[] Priority:0 State:retry_wait CreatedAt:2023-11-14 22:13:20 +0000 UTC UpdatedAt:2023-11-14 22:13:32.000000001 +0000 UTC AvailableAt:2023-11-14 22:13:37.000000001 +0000 UTC Deadline:<nil> Dependencies:[] MaxAttempts:3 Attempts:1 Backoff:{Kind:exponential BaseSeconds:5 MaxSeconds:3600 JitterPercent:10} RequiredLabels:map[] Resources:{CPU:0 MemoryMB:0 Slots:1} IdempotencyKey: Lease:<nil> Result:[] LastError:0x40000ca240 History:[{From: To:ready At:2023-11-14 22:13:20 +0000 UTC Reason:enqueued and ready Actor:enqueue} {From:ready To:leased At:2023-11-14 22:13:20 +0000 UTC Reason:claimed Actor:worker} {From:leased To:retry_wait At:2023-11-14 22:13:32.000000001 +0000 UTC Reason:retry scheduled in 5s Actor:worker}]}, <nil>), want nil job and expired-lease error
--- FAIL: TestFailRejectsLeaseAfterExpiryGrace (0.12s)
FAIL
FAIL	QueueForge/internal/engine	0.262s
FAIL

```

stderr：

```text
(empty)
```

## 通过条件

超过 heartbeat grace 的失败上报被拒绝，任务原 lease、状态、历史和错误字段保持不变；宽限期内的失败流程不回归；双架构定向、全量、build/vet 通过。
