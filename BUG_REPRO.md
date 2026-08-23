# Bug Reproduction

## 包的性质

当前 tasks/0006/green 保存的是被测模型修复后的结果源码，不是初始含 Bug 源码。最终 baseline 分支已在下面固定的 parent SHA 上单独提交任务测试，可直接复核 red；tasks/0006/green 包含同一测试并应得到 green。完整验证日志仍只在本地留存。

## 问题现象

卫星链路重放同一条遥测事件时，唯一键正确拒绝了第二次写入，但批次的事件数量却再次增加。值班员看到的总数大于实际明细数，无法可靠判断数据是否齐全。请修复重复上传处理，失败重放不得污染任何汇总，新的不同事件仍要正常累加。

## 含 Bug 版本

- 仓库：zhanglei10281852-gif/distant-water-fishing-ops-202608-root
- 仓库地址：https://github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root.git
- parent SHA：20b46a85bc1d476c1f4c7f420962133fd2e0ae1d

## 复现步骤

```bash
git clone -- https://github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root.git bug-repro
cd bug-repro
git checkout --detach 20b46a85bc1d476c1f4c7f420962133fd2e0ae1d
go test ./internal/runtimeops -run '^TestTask0006DuplicateTelemetryDoesNotPolluteBatch$' -count=1
```

## 双架构完整错误信息

### linux/amd64

- 容器内复现预期退出码：1
- 容器内复现实际退出码：1

stdout：

```text
$ go test ./internal/runtimeops -run '^TestTask0006DuplicateTelemetryDoesNotPolluteBatch$' -count=1
--- FAIL: TestTask0006DuplicateTelemetryDoesNotPolluteBatch (0.23s)
    task_0006_test.go:33: batch after duplicate=collecting count=2 err=<nil>
    task_0006_test.go:42: valid second event count=3 err=<nil>
FAIL
FAIL	github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/runtimeops	0.242s
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
$ go test ./internal/runtimeops -run '^TestTask0006DuplicateTelemetryDoesNotPolluteBatch$' -count=1
--- FAIL: TestTask0006DuplicateTelemetryDoesNotPolluteBatch (0.32s)
    task_0006_test.go:33: batch after duplicate=collecting count=2 err=<nil>
    task_0006_test.go:42: valid second event count=3 err=<nil>
FAIL
FAIL	github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/runtimeops	0.507s
FAIL

```

stderr：

```text
(empty)
```

## 通过条件

相同租户和事件 ID 的重放返回冲突后，事件明细仍只有一条且批次计数不变；随后写入不同事件时明细与计数同步增长到二。
