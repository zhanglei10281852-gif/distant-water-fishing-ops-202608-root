# BENZHI_README

## 项目说明

- 项目：zhanglei10281852-gif/distant-water-fishing-ops-202608-root
- 项目用途：Distant-Water Fishing Operations is a production-oriented Go backend for planning distant-water fishing voyages, coordinating departure and landing ports, reserving licensed vessels and support fleets, reconciling catch declarations, and completing post-landing fisheries compliance review. It uses a durable SQLite workflow and does not depend on online services.
- Go 工具链：`golang:1.22`
- 前端工具链：无

## 标准构建、运行和测试命令

进入容器后执行：

```bash
# 编译
cd '/app' && GOTOOLCHAIN=local go build ./...

# 启动
cd '/app' && GOTOOLCHAIN=local go run ./cmd/seed-user
cd '/app' && GOTOOLCHAIN=local go run ./cmd/server

# 测试
cd '/app' && GOTOOLCHAIN=local go test ./...
```

## Docker 构建和进入容器

```bash
chmod +x build_benzhi_docker.sh
./build_benzhi_docker.sh benzhi-task-130-amd64 linux/amd64
./build_benzhi_docker.sh benzhi-task-130-arm64 linux/arm64
docker run -it benzhi-task-130-amd64:latest
docker run -it --platform linux/arm64 benzhi-task-130-arm64:latest
```

## 题目验证命令

1. 预期退出码 1：`go test ./internal/runtimeops -run '^TestTask0010TelemetryRowsStayInsideFleetBoundary$' -count=1`

## Bug 复现

Bug 现象、触发步骤和完整错误信息见 `BUG_REPRO.md`。
