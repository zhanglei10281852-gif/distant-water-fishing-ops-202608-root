# Bug Reproduction

## 包的性质

当前 tasks/0002/green 保存 diagnosis 的最终交付提交：生产源码与下面固定的 parent SHA 完全一致，不包含模型修复；该提交只增加与 red R1 相同的任务测试和交付文件，运行题目验证命令仍应得到 red。完整验证日志仍只在本地留存。

## 问题现象

两名审核员先后处理同一张远洋作业许可，后到请求携带旧版本并被系统拒绝，但审计列表里仍出现了一条 approval 记录，值班人员误以为两次审批都成功。先不要修改代码，请复现这种版本冲突，查明为什么失败请求仍能制造持久化审计。

## 含 Bug 版本

- 仓库：zhanglei10281852-gif/distant-water-fishing-ops-202608-root
- 仓库地址：https://github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root.git
- parent SHA：d24973487c7429bc9574f8183aa3509498366a25

## 复现步骤

```bash
git clone -- https://github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root.git bug-repro
cd bug-repro
git checkout --detach d24973487c7429bc9574f8183aa3509498366a25
go test ./internal/runtimeops -run '^TestTask0002VersionConflictLeavesNoApprovalAudit$' -count=1
```

## 双架构完整错误信息

### linux/amd64

- 容器内复现预期退出码：1
- 容器内复现实际退出码：1

stdout：

```text
$ go test ./internal/runtimeops -run '^TestTask0002VersionConflictLeavesNoApprovalAudit$' -count=1
--- FAIL: TestTask0002VersionConflictLeavesNoApprovalAudit (0.15s)
    task_0002_test.go:35: stale approval audit count=1 err=<nil>
    task_0002_test.go:44: valid approval audit count=2 err=<nil>
FAIL
FAIL	github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/runtimeops	0.154s
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
$ go test ./internal/runtimeops -run '^TestTask0002VersionConflictLeavesNoApprovalAudit$' -count=1
--- FAIL: TestTask0002VersionConflictLeavesNoApprovalAudit (0.30s)
    task_0002_test.go:35: stale approval audit count=1 err=<nil>
    task_0002_test.go:44: valid approval audit count=2 err=<nil>
FAIL
FAIL	github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/runtimeops	0.481s
FAIL

```

stderr：

```text
(empty)
```

## 通过条件

诊断结论需指向许可审批实现及具体方法，解释审计与乐观版本更新分属不同事务边界后，旧版本更新返回冲突而先写审计无法回滚的完整机制，并用许可行与审计行证据佐证。诊断结论和证据交付后，目标仓库代码保持零改动。
