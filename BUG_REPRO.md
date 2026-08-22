# Bug Reproduction

## 包的性质

当前 tasks/0018/green 保存 diagnosis 的最终交付提交：生产源码与下面固定的 parent SHA 完全一致，不包含模型修复；该提交只增加与 red R1 相同的任务测试和交付文件，运行题目验证命令仍应得到 red。完整验证日志仍只在本地留存。

## 问题现象

船队 A 发起过期租约回收后，船队 B 的运行中作业也被改成 failed。B 的 worker 仍在正常处理，随后无法提交结果。先不要修改代码，请查明回收扫描和更新为何跨越船队隔离边界。

## 含 Bug 版本

- 仓库：zhanglei10281852-gif/distant-water-fishing-ops-202608-root
- 仓库地址：https://github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root.git
- parent SHA：e7b618fd2a31b750f0704aa13b3460215902c0d7

## 复现步骤

```bash
git clone -- https://github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root.git bug-repro
cd bug-repro
git checkout --detach e7b618fd2a31b750f0704aa13b3460215902c0d7
go test ./internal/runtimeops -run '^TestTask0018LeaseSweepIsScopedToFleet$' -count=1
```

## 双架构完整错误信息

### linux/amd64

- 容器内复现预期退出码：1
- 容器内复现实际退出码：1

stdout：

```text
$ go test ./internal/runtimeops -run '^TestTask0018LeaseSweepIsScopedToFleet$' -count=1
--- FAIL: TestTask0018LeaseSweepIsScopedToFleet (0.14s)
    task_0018_test.go:35: fleet-a reclaim count=2 err=<nil>
    task_0018_test.go:43: job-b={ID:job-b TenantID:fleet-b State:failed Payload:x Attempts:1 MaxAttempts:3 AvailableAt:2026-08-22 09:00:00 +0000 UTC LeaseUntil:<nil>} err=<nil>
    task_0018_test.go:47: fleet-b reclaim count=0 err=<nil>
FAIL
FAIL	github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/runtimeops	0.142s
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
$ go test ./internal/runtimeops -run '^TestTask0018LeaseSweepIsScopedToFleet$' -count=1
--- FAIL: TestTask0018LeaseSweepIsScopedToFleet (0.40s)
    task_0018_test.go:35: fleet-a reclaim count=2 err=<nil>
    task_0018_test.go:43: job-b={ID:job-b TenantID:fleet-b State:failed Payload:x Attempts:1 MaxAttempts:3 AvailableAt:2026-08-22 09:00:00 +0000 UTC LeaseUntil:<nil>} err=<nil>
    task_0018_test.go:47: fleet-b reclaim count=0 err=<nil>
FAIL
FAIL	github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/runtimeops	0.619s
FAIL

```

stderr：

```text
(empty)
```

## 通过条件

诊断要定位租约回收实现和具体符号，说明全局扫描收集所有租户的过期作业并按其原租户逐条更新，导致调用参数失效；证据包括 A 被回收、B 保持 running 以及 B 自己回收的合法路径。诊断结论验收只看分析材料，目标仓库代码保持零改动。
