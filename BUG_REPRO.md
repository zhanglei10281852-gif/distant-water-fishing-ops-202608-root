# Bug Reproduction

## 包的性质

当前 tasks/0028/green 保存的是被测模型修复后的结果源码，不是初始含 Bug 源码。最终 baseline 分支已在下面固定的 parent SHA 上单独提交任务测试，可直接复核 red；tasks/0028/green 包含同一测试并应得到 green。完整验证日志仍只在本地留存。

## 问题现象

上游在重试间隔开始后立即取消请求，服务仍完整等待三百毫秒才退出，关停过程被大量退避任务拖慢。请修复退避等待，使取消可以即时打断计时，同时保留短暂故障后再次尝试并成功的行为。

## 含 Bug 版本

- 仓库：zhanglei10281852-gif/distant-water-fishing-ops-202608-root
- 仓库地址：https://github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root.git
- parent SHA：daff2768674bb146ee6790713fe0c1745a329b1e

## 复现步骤

```bash
git clone -- https://github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root.git bug-repro
cd bug-repro
git checkout --detach daff2768674bb146ee6790713fe0c1745a329b1e
go test ./internal/runtimeops -run '^TestTask0028CancellationInterruptsRetryBackoff$' -count=1
```

## 双架构完整错误信息

### linux/amd64

- 容器内复现预期退出码：1
- 容器内复现实际退出码：1

stdout：

```text
$ go test ./internal/runtimeops -run '^TestTask0028CancellationInterruptsRetryBackoff$' -count=1
--- FAIL: TestTask0028CancellationInterruptsRetryBackoff (0.30s)
    task_0028_test.go:41: backoff ignored cancellation for 300.486104ms
FAIL
FAIL	github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/runtimeops	0.305s
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
$ go test ./internal/runtimeops -run '^TestTask0028CancellationInterruptsRetryBackoff$' -count=1
--- FAIL: TestTask0028CancellationInterruptsRetryBackoff (0.31s)
    task_0028_test.go:41: backoff ignored cancellation for 304.505986ms
FAIL
FAIL	github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/runtimeops	0.533s
FAIL

```

stderr：

```text
(empty)
```

## 通过条件

取消发生在首次失败后时只执行一次 operation，并在退避时长之前返回 context cancellation；未取消场景仍按配置等待并在第二次调用成功。
