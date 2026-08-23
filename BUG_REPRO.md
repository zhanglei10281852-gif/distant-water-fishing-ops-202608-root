# Bug Reproduction

## 包的性质

当前 tasks/0021/green 保存的是被测模型修复后的结果源码，不是初始含 Bug 源码。最终 baseline 分支已在下面固定的 parent SHA 上单独提交任务测试，可直接复核 red；tasks/0021/green 包含同一测试并应得到 green。完整验证日志仍只在本地留存。

## 问题现象

岸端收到相同幂等键但载荷已经变化的重试请求时，系统仍回放第一次的成功响应，调用方以为新内容已经处理。请修复回放校验，只有请求摘要一致时才能复用响应，并继续保证返回字节不会被调用方修改后污染存储。

## 含 Bug 版本

- 仓库：zhanglei10281852-gif/distant-water-fishing-ops-202608-root
- 仓库地址：https://github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root.git
- parent SHA：10312a33a137fc9dbf1f12b9178f7332d05895bb

## 复现步骤

```bash
git clone -- https://github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root.git bug-repro
cd bug-repro
git checkout --detach 10312a33a137fc9dbf1f12b9178f7332d05895bb
go test ./internal/runtimeops -run '^TestTask0021ReplayRejectsDifferentRequestPayload$' -count=1
```

## 双架构完整错误信息

### linux/amd64

- 容器内复现预期退出码：1
- 容器内复现实际退出码：1

stdout：

```text
$ go test ./internal/runtimeops -run '^TestTask0021ReplayRejectsDifferentRequestPayload$' -count=1
--- FAIL: TestTask0021ReplayRejectsDifferentRequestPayload (0.10s)
    task_0021_test.go:28: mismatched replay body="accepted-v1" err=<nil>
FAIL
FAIL	github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/runtimeops	0.100s
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
$ go test ./internal/runtimeops -run '^TestTask0021ReplayRejectsDifferentRequestPayload$' -count=1
--- FAIL: TestTask0021ReplayRejectsDifferentRequestPayload (0.32s)
    task_0021_test.go:28: mismatched replay body="accepted-v1" err=<nil>
FAIL
FAIL	github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/runtimeops	0.506s
FAIL

```

stderr：

```text
(empty)
```

## 通过条件

不同 request hash 的回放返回冲突且无响应体；匹配摘要得到原响应，调用方修改返回切片后再次回放仍读取未污染的持久化内容。
