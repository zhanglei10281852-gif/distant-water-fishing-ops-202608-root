# Bug Reproduction

## 包的性质

当前 tasks/0015/green 保存 diagnosis 的最终交付提交：生产源码与下面固定的 parent SHA 完全一致，不包含模型修复；该提交只增加与 red R1 相同的任务测试和交付文件，运行题目验证命令仍应得到 red。完整验证日志仍只在本地留存。

## 问题现象

旧 worker 的租约已经过期并被回收，新 worker 领取后 attempt 变为二；此时旧 worker 用 attempt 一上报完成，系统竟接受并把新租约直接置为 done。先不要修改代码，请定位完成确认为什么没有验证调用方仍持有当前租约。

## 含 Bug 版本

- 仓库：zhanglei10281852-gif/distant-water-fishing-ops-202608-root
- 仓库地址：https://github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root.git
- parent SHA：e711b689d0e500a81923d85ed831fb59284946fa

## 复现步骤

```bash
git clone -- https://github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root.git bug-repro
cd bug-repro
git checkout --detach e711b689d0e500a81923d85ed831fb59284946fa
go test ./internal/runtimeops -run '^TestTask0015ExpiredWorkerCannotCompleteNewLease$' -count=1
```

## 双架构完整错误信息

### linux/amd64

- 容器内复现预期退出码：1
- 容器内复现实际退出码：1

stdout：

```text
$ go test ./internal/runtimeops -run '^TestTask0015ExpiredWorkerCannotCompleteNewLease$' -count=1
--- FAIL: TestTask0015ExpiredWorkerCannotCompleteNewLease (0.15s)
    task_0015_test.go:40: stale completion error=<nil>
    task_0015_test.go:44: job after stale worker={ID:leased TenantID:fleet-o State:done Payload:sample Attempts:2 MaxAttempts:4 AvailableAt:2026-08-22 06:00:00 +0000 UTC LeaseUntil:<nil>} err=<nil>
    task_0015_test.go:46: not found
FAIL
FAIL	github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/runtimeops	0.159s
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
$ go test ./internal/runtimeops -run '^TestTask0015ExpiredWorkerCannotCompleteNewLease$' -count=1
--- FAIL: TestTask0015ExpiredWorkerCannotCompleteNewLease (0.42s)
    task_0015_test.go:40: stale completion error=<nil>
    task_0015_test.go:44: job after stale worker={ID:leased TenantID:fleet-o State:done Payload:sample Attempts:2 MaxAttempts:4 AvailableAt:2026-08-22 06:00:00 +0000 UTC LeaseUntil:<nil>} err=<nil>
    task_0015_test.go:46: not found
FAIL
FAIL	github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/runtimeops	0.651s
FAIL

```

stderr：

```text
(empty)
```

## 通过条件

诊断需指出作业完成方法及其持久化更新，解释读取当前 attempt 后却在 UPDATE 中移除 attempt 条件，使旧持有者可以覆盖新租约；复现证据应展示旧调用被拒绝前后的状态及新持有者合法完成。诊断结论必须在不改代码的前提下形成，并保留原始故障现场。
