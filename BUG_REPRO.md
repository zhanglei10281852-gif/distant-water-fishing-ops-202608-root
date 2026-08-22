# Bug Reproduction

## 包的性质

当前 tasks/0017/green 保存的是被测模型修复后的结果源码，不是初始含 Bug 源码。最终 baseline 分支已在下面固定的 parent SHA 上单独提交任务测试，可直接复核 red；tasks/0017/green 包含同一测试并应得到 green。完整验证日志仍只在本地留存。

## 问题现象

运维停止一次过期租约清理后，数据库里仍有作业被改成 failed，说明清理动作脱离了调用方生命周期。请修复回收流程，让取消信号贯穿数据库事务；未取消的清理仍应准确回收已过期租约。

## 含 Bug 版本

- 仓库：zhanglei10281852-gif/distant-water-fishing-ops-202608-root
- 仓库地址：https://github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root.git
- parent SHA：9532e82c9eea603b9ad43b4452b87273bfb552d6

## 复现步骤

```bash
git clone -- https://github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root.git bug-repro
cd bug-repro
git checkout --detach 9532e82c9eea603b9ad43b4452b87273bfb552d6
go test ./internal/runtimeops -run '^TestTask0017CancelledLeaseSweepMakesNoChanges$' -count=1
```

## 双架构完整错误信息

### linux/amd64

- 容器内复现预期退出码：1
- 容器内复现实际退出码：1

stdout：

```text
$ go test ./internal/runtimeops -run '^TestTask0017CancelledLeaseSweepMakesNoChanges$' -count=1
--- FAIL: TestTask0017CancelledLeaseSweepMakesNoChanges (0.19s)
    task_0017_test.go:33: cancelled reclaim count=1 err=<nil>
    task_0017_test.go:37: job after cancelled sweep={ID:sweep TenantID:fleet-q State:failed Payload:x Attempts:1 MaxAttempts:3 AvailableAt:2026-08-22 08:00:00 +0000 UTC LeaseUntil:<nil>} err=<nil>
    task_0017_test.go:41: valid reclaim count=0 err=<nil>
FAIL
FAIL	github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/runtimeops	0.189s
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
$ go test ./internal/runtimeops -run '^TestTask0017CancelledLeaseSweepMakesNoChanges$' -count=1
--- FAIL: TestTask0017CancelledLeaseSweepMakesNoChanges (0.29s)
    task_0017_test.go:33: cancelled reclaim count=1 err=<nil>
    task_0017_test.go:37: job after cancelled sweep={ID:sweep TenantID:fleet-q State:failed Payload:x Attempts:1 MaxAttempts:3 AvailableAt:2026-08-22 08:00:00 +0000 UTC LeaseUntil:<nil>} err=<nil>
    task_0017_test.go:41: valid reclaim count=0 err=<nil>
FAIL
FAIL	github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/runtimeops	0.465s
FAIL

```

stderr：

```text
(empty)
```

## 通过条件

取消后的回收影响零行并返回 context 错误，作业保持 running 和 lease；正常调用只回收当前船队已过期的作业并返回真实影响数。
