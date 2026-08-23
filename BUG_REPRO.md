# Bug Reproduction

## 包的性质

当前 tasks/0008/green 保存的是被测模型修复后的结果源码，不是初始含 Bug 源码。最终 baseline 分支已在下面固定的 parent SHA 上单独提交任务测试，可直接复核 red；tasks/0008/green 包含同一测试并应得到 green。完整验证日志仍只在本地留存。

## 问题现象

遥测批次仍含未分类事件时，关闭请求按预期返回冲突，但刷新页面后批次已经是 closed，分类员再也无法完成剩余处置。请修复关闭顺序，拒绝路径不能提前改变批次状态；所有事件完成分类后才允许真正关闭。

## 含 Bug 版本

- 仓库：zhanglei10281852-gif/distant-water-fishing-ops-202608-root
- 仓库地址：https://github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root.git
- parent SHA：46fe96551a14899a6b09ffec66c1ee3615c72ebb

## 复现步骤

```bash
git clone -- https://github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root.git bug-repro
cd bug-repro
git checkout --detach 46fe96551a14899a6b09ffec66c1ee3615c72ebb
go test ./internal/runtimeops -run '^TestTask0008RejectedClosureKeepsBatchCollecting$' -count=1
```

## 双架构完整错误信息

### linux/amd64

- 容器内复现预期退出码：1
- 容器内复现实际退出码：1

stdout：

```text
$ go test ./internal/runtimeops -run '^TestTask0008RejectedClosureKeepsBatchCollecting$' -count=1
--- FAIL: TestTask0008RejectedClosureKeepsBatchCollecting (0.24s)
    task_0008_test.go:32: rejected closure state=closed count=1 err=<nil>
    task_0008_test.go:35: conflict
FAIL
FAIL	github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/runtimeops	0.251s
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
$ go test ./internal/runtimeops -run '^TestTask0008RejectedClosureKeepsBatchCollecting$' -count=1
--- FAIL: TestTask0008RejectedClosureKeepsBatchCollecting (0.43s)
    task_0008_test.go:32: rejected closure state=closed count=1 err=<nil>
    task_0008_test.go:35: conflict
FAIL
FAIL	github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/runtimeops	0.750s
FAIL

```

stderr：

```text
(empty)
```

## 通过条件

存在未分类事件时返回冲突且批次保持 collecting、计数与事件不变；分类完成后的再次关闭成功进入 closed，并保留原事件数量。
