# BENZHI_README

这是一个面向远洋渔业作业合规管理的 Go 后端服务，负责渔船许可、航次调度、港口协同、渔获申报、靠港核验和审计通知。

## 构建、运行和测试

```bash
cd '/app' && GOTOOLCHAIN=local go build ./...
cd '/app' && GOTOOLCHAIN=local go run ./cmd/seed-user
cd '/app' && GOTOOLCHAIN=local go run ./cmd/server
cd '/app' && GOTOOLCHAIN=local go test ./...
```

## Docker 构建和运行

```bash
chmod +x build_benzhi_docker.sh
./build_benzhi_docker.sh benzhi-task-121-amd64 linux/amd64
./build_benzhi_docker.sh benzhi-task-121-arm64 linux/arm64
docker run -it benzhi-task-121-amd64:latest
docker run -it --platform linux/arm64 benzhi-task-121-arm64:latest
```
