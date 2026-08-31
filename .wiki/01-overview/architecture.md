# 架构

cst-cli 是单进程本地 CLI：Cobra 解析参数，Bubble Tea 跑全屏 TUI，领域逻辑在 `internal/` 包中，通过本机进程或 SSH 访问外部工具。

## 分层

```text
main.go
  └── cmd/*          Cobra：flags、别名、调用 tui.Run*
        └── internal/tui/*     交互状态机与绘制
              ├── internal/maven     发现 pom、执行 mvn
              ├── internal/jars      发现 / 过滤 / 拷贝 / 清空 jar
              ├── internal/deploy    读取 deploy.yaml
              ├── internal/upload    读取 servers.yaml、SFTP、串行 docker restart
              ├── internal/docker    远程 docker ps / restart / logs
              └── internal/git       发现仓库、status、变更树
```

TUI 不反向被领域包依赖。`internal/maven` 注释写明「无 UI 依赖」。

## 进程内协作

- 长任务（构建、上传、跟日志）在 goroutine 里跑，通过 `chan tea.Msg` 把进度打回 Bubble Tea `Update`
- `deploy` 上传：每个 jar 各自 SSH+SFTP（[internal/upload/upload.go](../../internal/upload/upload.go) `UploadAll`）
- `deploy` 重启：一条 SSH，容器串行 `docker restart`，间隔 `RestartPause`（5s）
- `docker` 重启：一条 SSH，组内并行 `RestartAndFollow`，每台容器独立日志面板

## 配置与密钥边界

连接信息只来自本机 YAML（默认 `~/.config/cst-cli/`）。仓库不提交真实 `servers.yaml`。SSH 使用密码与 keyboard-interactive，且 `HostKeyCallback` 为 `ssh.InsecureIgnoreHostKey()`（[internal/upload/upload.go](../../internal/upload/upload.go) `dial`）。

## 非目标

- 不是远程 agent / HTTP 网关
- 不管理 Docker Compose 或 K8s
- 不实现 Maven 本身，只调用本机 `mvn`

## Related

- Code: [main.go](../../main.go), [cmd/root.go](../../cmd/root.go)
- Modules: [module-map.md](module-map.md)
- Integrations: [../08-external-integrations/integrations.md](../08-external-integrations/integrations.md)
