# Bug Reproduction

## 包的性质

当前 tasks/0010/green 保存 diagnosis 的最终交付提交：生产源码与下面固定的 parent SHA 完全一致，不包含模型修复；该提交只增加与 red R1 相同的任务测试和交付文件，运行题目验证命令仍应得到 red。完整验证日志仍只在本地留存。

## 问题现象

监管账号查看指定船队的已分类遥测时，响应 total 是一条，items 却夹带了另一船队的事件内容，构成跨租户数据泄漏。先不要修改代码，请复现并定位分页明细读取为何绕过了船队边界。

## 含 Bug 版本

- 仓库：zhanglei10281852-gif/distant-water-fishing-ops-202608-root
- 仓库地址：https://github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root.git
- parent SHA：1e99268dce64a2f71731ff39ffb8d952a341bceb

## 复现步骤

```bash
git clone -- https://github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root.git bug-repro
cd bug-repro
git checkout --detach 1e99268dce64a2f71731ff39ffb8d952a341bceb
go test ./internal/runtimeops -run '^TestTask0010TelemetryRowsStayInsideFleetBoundary$' -count=1
```

## 双架构完整错误信息

### linux/amd64

- 容器内复现预期退出码：1
- 容器内复现实际退出码：1

stdout：

```text
$ go test ./internal/runtimeops -run '^TestTask0010TelemetryRowsStayInsideFleetBoundary$' -count=1
--- FAIL: TestTask0010TelemetryRowsStayInsideFleetBoundary (0.18s)
    task_0010_test.go:34: fleet-a page={Items:[{ID:a-event TenantID:fleet-a BatchID:items-a Status:classified Magnitude:1} {ID:b-event TenantID:fleet-b BatchID:items-b Status:classified Magnitude:2}] Total:1}
    task_0010_test.go:38: foreign event leaked: {ID:b-event TenantID:fleet-b BatchID:items-b Status:classified Magnitude:2}
FAIL
FAIL	github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/runtimeops	0.178s
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
$ go test ./internal/runtimeops -run '^TestTask0010TelemetryRowsStayInsideFleetBoundary$' -count=1
--- FAIL: TestTask0010TelemetryRowsStayInsideFleetBoundary (0.32s)
    task_0010_test.go:34: fleet-a page={Items:[{ID:a-event TenantID:fleet-a BatchID:items-a Status:classified Magnitude:1} {ID:b-event TenantID:fleet-b BatchID:items-b Status:classified Magnitude:2}] Total:1}
    task_0010_test.go:38: foreign event leaked: {ID:b-event TenantID:fleet-b BatchID:items-b Status:classified Magnitude:2}
FAIL
FAIL	github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/runtimeops	0.510s
FAIL

```

stderr：

```text
(empty)
```

## 通过条件

诊断应指出事件分页实现及具体符号，解释总数查询保留租户条件而明细查询另建无租户条件后产生 total/items 分裂和数据泄漏；还要用空状态过滤的合法结果作对照。诊断结论以只读排查结束，目标仓库代码保持零改动。
