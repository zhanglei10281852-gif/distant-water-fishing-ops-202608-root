# Bug Reproduction

## 包的性质

当前 tasks/0025/green 保存的是被测模型修复后的结果源码，不是初始含 Bug 源码。最终 baseline 分支已在下面固定的 parent SHA 上单独提交任务测试，可直接复核 red；tasks/0025/green 包含同一测试并应得到 green。完整验证日志仍只在本地留存。

## 问题现象

监控组件读取流水线快照后，为页面展示修改了返回 map，运行中的归档流程随即把未执行步骤视为完成、已完成步骤也可能被重新执行。请修复快照所有权，调用方只能修改自己的副本，不能污染内部状态。

## 含 Bug 版本

- 仓库：zhanglei10281852-gif/distant-water-fishing-ops-202608-root
- 仓库地址：https://github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root.git
- parent SHA：3b1f1efe47ea2678430791423e6eeb5a22fcbac5

## 复现步骤

```bash
git clone -- https://github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root.git bug-repro
cd bug-repro
git checkout --detach 3b1f1efe47ea2678430791423e6eeb5a22fcbac5
go test ./internal/runtimeops -run '^TestTask0025SnapshotMutationCannotChangePipelineState$' -count=1
```

## 双架构完整错误信息

### linux/amd64

- 容器内复现预期退出码：1
- 容器内复现实际退出码：1

stdout：

```text
$ go test ./internal/runtimeops -run '^TestTask0025SnapshotMutationCannotChangePipelineState$' -count=1
--- FAIL: TestTask0025SnapshotMutationCannotChangePipelineState (0.00s)
    task_0025_test.go:33: pipeline polluted by snapshot mutation: map[archive:false invented:true]
    task_0025_test.go:37: completed step reran 2 times
FAIL
FAIL	github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/runtimeops	0.003s
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
$ go test ./internal/runtimeops -run '^TestTask0025SnapshotMutationCannotChangePipelineState$' -count=1
--- FAIL: TestTask0025SnapshotMutationCannotChangePipelineState (0.01s)
    task_0025_test.go:33: pipeline polluted by snapshot mutation: map[archive:false invented:true]
    task_0025_test.go:37: completed step reran 2 times
FAIL
FAIL	github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/runtimeops	0.220s
FAIL

```

stderr：

```text
(empty)
```

## 通过条件

修改返回快照中的已有键或新增键都不改变流水线内部 map；已成功步骤保持完成且后续运行不会重复调用，快照仍准确反映内部状态。
