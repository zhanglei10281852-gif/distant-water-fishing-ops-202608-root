# Bug Reproduction

## 包的性质

当前 tasks/0005/green 保存 diagnosis 的最终交付提交：生产源码与下面固定的 parent SHA 完全一致，不包含模型修复；该提交只增加与 red R1 相同的任务测试和交付文件，运行题目验证命令仍应得到 red。完整验证日志仍只在本地留存。

## 问题现象

船载遥测上传一条观测后，批次汇总更新因数据库锁冲突失败。接口返回错误，但事件列表仍能查到这条观测，而批次计数保持为零，后续结批得到互相矛盾的数据。先不要修改代码，请复现并定位事件明细为何没有随着汇总失败一起回滚。

## 含 Bug 版本

- 仓库：zhanglei10281852-gif/distant-water-fishing-ops-202608-root
- 仓库地址：https://github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root.git
- parent SHA：d6c0799e1cc0c4154a261d600fd1ff544a145c10

## 复现步骤

```bash
git clone -- https://github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root.git bug-repro
cd bug-repro
git checkout --detach d6c0799e1cc0c4154a261d600fd1ff544a145c10
go test ./internal/runtimeops -run '^TestTask0005TelemetryWriteRollsBackWhenBatchUpdateFails$' -count=1
```

## 双架构完整错误信息

### linux/amd64

- 容器内复现预期退出码：1
- 容器内复现实际退出码：1

stdout：

```text
$ go test ./internal/runtimeops -run '^TestTask0005TelemetryWriteRollsBackWhenBatchUpdateFails$' -count=1
--- FAIL: TestTask0005TelemetryWriteRollsBackWhenBatchUpdateFails (0.13s)
    task_0005_test.go:38: events={Items:[{ID:event-failed TenantID:fleet-e BatchID:batch-e Status:unclassified Magnitude:7.4}] Total:1} err=<nil>
FAIL
FAIL	github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/runtimeops	0.135s
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
$ go test ./internal/runtimeops -run '^TestTask0005TelemetryWriteRollsBackWhenBatchUpdateFails$' -count=1
--- FAIL: TestTask0005TelemetryWriteRollsBackWhenBatchUpdateFails (0.43s)
    task_0005_test.go:38: events={Items:[{ID:event-failed TenantID:fleet-e BatchID:batch-e Status:unclassified Magnitude:7.4}] Total:1} err=<nil>
FAIL
FAIL	github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/runtimeops	0.655s
FAIL

```

stderr：

```text
(empty)
```

## 通过条件

诊断必须命中遥测写入所在 Go 文件和具体方法，说明事件 INSERT 在事务外提交、批次 UPDATE 在另一事务失败后无法撤销前者的因果链，并以事件集合、批次状态和合法写入对照作为证据。诊断结论形成前后均不修改生产代码或配置。
