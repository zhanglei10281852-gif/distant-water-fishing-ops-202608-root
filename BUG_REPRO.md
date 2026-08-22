# Bug Reproduction

## 包的性质

当前 tasks/0009/green 保存的是被测模型修复后的结果源码，不是初始含 Bug 源码。最终 baseline 分支已在下面固定的 parent SHA 上单独提交任务测试，可直接复核 red；tasks/0009/green 包含同一测试并应得到 green。完整验证日志仍只在本地留存。

## 问题现象

船队 A 的遥测列表只展示自己的一条记录，分页元数据却把船队 B 的两条记录也算进总数，前端因此生成了不存在的后续页。请修复分页统计，让 total 与同一租户、同一状态过滤下的 items 使用完全一致的数据范围。

## 含 Bug 版本

- 仓库：zhanglei10281852-gif/distant-water-fishing-ops-202608-root
- 仓库地址：https://github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root.git
- parent SHA：a2651039245dcc6461ad82050b6a683c1fae1e78

## 复现步骤

```bash
git clone -- https://github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root.git bug-repro
cd bug-repro
git checkout --detach a2651039245dcc6461ad82050b6a683c1fae1e78
go test ./internal/runtimeops -run '^TestTask0009TelemetryPageTotalMatchesTenantItems$' -count=1
```

## 双架构完整错误信息

### linux/amd64

- 容器内复现预期退出码：1
- 容器内复现实际退出码：1

stdout：

```text
$ go test ./internal/runtimeops -run '^TestTask0009TelemetryPageTotalMatchesTenantItems$' -count=1
--- FAIL: TestTask0009TelemetryPageTotalMatchesTenantItems (0.24s)
    task_0009_test.go:32: fleet-a page={Items:[{ID:a-one TenantID:fleet-a BatchID:page-a Status:classified Magnitude:1}] Total:3} err=<nil>
FAIL
FAIL	github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/runtimeops	0.239s
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
$ go test ./internal/runtimeops -run '^TestTask0009TelemetryPageTotalMatchesTenantItems$' -count=1
--- FAIL: TestTask0009TelemetryPageTotalMatchesTenantItems (0.29s)
    task_0009_test.go:32: fleet-a page={Items:[{ID:a-one TenantID:fleet-a BatchID:page-a Status:classified Magnitude:1}] Total:3} err=<nil>
FAIL
FAIL	github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/runtimeops	0.453s
FAIL

```

stderr：

```text
(empty)
```

## 通过条件

无过滤和状态过滤两种查询中，total 都与当前船队实际返回项一致，异租户事件不进入统计；合法过滤仍能准确找到目标状态记录。
