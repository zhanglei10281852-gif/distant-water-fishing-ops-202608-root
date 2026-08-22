# Bug Reproduction

## 包的性质

当前 tasks/0026/green 保存的是被测模型修复后的结果源码，不是初始含 Bug 源码。最终 baseline 分支已在下面固定的 parent SHA 上单独提交任务测试，可直接复核 red；tasks/0026/green 包含同一测试并应得到 green。完整验证日志仍只在本地留存。

## 问题现象

服务从持久化快照恢复航次流水线后，上层代码继续复用并修改传入的 map，流水线状态也跟着变化，造成步骤无故跳过。请修复恢复过程的所有权边界，对外部输入做独立持有，同时保留 false 状态的正常恢复语义。

## 含 Bug 版本

- 仓库：zhanglei10281852-gif/distant-water-fishing-ops-202608-root
- 仓库地址：https://github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root.git
- parent SHA：12ebe463674947b46648a09f91853330b0a8bd60

## 复现步骤

```bash
git clone -- https://github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root.git bug-repro
cd bug-repro
git checkout --detach 12ebe463674947b46648a09f91853330b0a8bd60
go test ./internal/runtimeops -run '^TestTask0026RestoreDoesNotRetainCallerOwnedMap$' -count=1
```

## 双架构完整错误信息

### linux/amd64

- 容器内复现预期退出码：1
- 容器内复现实际退出码：1

stdout：

```text
$ go test ./internal/runtimeops -run '^TestTask0026RestoreDoesNotRetainCallerOwnedMap$' -count=1
--- FAIL: TestTask0026RestoreDoesNotRetainCallerOwnedMap (0.00s)
    task_0026_test.go:31: restored state followed caller mutation: map[foreign-step:true sync-manifest:false]
    task_0026_test.go:36: restored completed step reran 1 times
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
$ go test ./internal/runtimeops -run '^TestTask0026RestoreDoesNotRetainCallerOwnedMap$' -count=1
--- FAIL: TestTask0026RestoreDoesNotRetainCallerOwnedMap (0.01s)
    task_0026_test.go:31: restored state followed caller mutation: map[foreign-step:true sync-manifest:false]
    task_0026_test.go:36: restored completed step reran 1 times
FAIL
FAIL	github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/runtimeops	0.186s
FAIL

```

stderr：

```text
(empty)
```

## 通过条件

Restore 后修改调用方 map 不影响内部完成集合；恢复为已完成的步骤不再执行，重新恢复包含 false 的新快照时内部值与输入时刻一致而不跟随后续修改。
