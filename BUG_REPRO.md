# Bug Reproduction

## 包的性质

当前 tasks/0019/green 保存的是被测模型修复后的结果源码，不是初始含 Bug 源码。最终 baseline 分支已在下面固定的 parent SHA 上单独提交任务测试，可直接复核 red；tasks/0019/green 包含同一测试并应得到 green。完整验证日志仍只在本地留存。

## 问题现象

两个船队通过卫星链路使用了相同幂等键，但请求内容和响应完全不同。船队 B 的首次提交被当成船队 A 的重放冲突，无法创建自己的业务。请修复幂等记录作用域，使租户隔离参与查询与持久化，同时同租户同操作的不同载荷仍要拒绝。

## 含 Bug 版本

- 仓库：zhanglei10281852-gif/distant-water-fishing-ops-202608-root
- 仓库地址：https://github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root.git
- parent SHA：d450ee3152f21c38f0ece5f3fc2700f7b50b3ded

## 复现步骤

```bash
git clone -- https://github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root.git bug-repro
cd bug-repro
git checkout --detach d450ee3152f21c38f0ece5f3fc2700f7b50b3ded
go test ./internal/runtimeops -run '^TestTask0019IdempotencyKeysAreTenantScoped$' -count=1
```

## 双架构完整错误信息

### linux/amd64

- 容器内复现预期退出码：1
- 容器内复现实际退出码：1

stdout：

```text
$ go test ./internal/runtimeops -run '^TestTask0019IdempotencyKeysAreTenantScoped$' -count=1
--- FAIL: TestTask0019IdempotencyKeysAreTenantScoped (0.10s)
    task_0019_test.go:29: second tenant save failed: conflict
    task_0019_test.go:34: replay fleet-b="" err=not found
FAIL
FAIL	github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/runtimeops	0.104s
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
$ go test ./internal/runtimeops -run '^TestTask0019IdempotencyKeysAreTenantScoped$' -count=1
--- FAIL: TestTask0019IdempotencyKeysAreTenantScoped (0.44s)
    task_0019_test.go:29: second tenant save failed: conflict
    task_0019_test.go:34: replay fleet-b="" err=not found
FAIL
FAIL	github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/runtimeops	0.648s
FAIL

```

stderr：

```text
(empty)
```

## 通过条件

相同 key 可由不同船队分别保存和回放各自响应；同一船队、方法和路径上的载荷变化继续返回冲突，不覆盖已保存结果。
