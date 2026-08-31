# 仓库结构

单模块 Go 仓库，无前端、无 `docs/`、无 `.wiki/` 以外的文档树（本 Wiki 为后补）。

```text
cst-cli/
├── main.go                 进程入口
├── go.mod / go.sum
├── cmd/                    Cobra 子命令
│   ├── root.go             Execute、注册命令、Version
│   ├── mvn.go
│   ├── deploy.go
│   ├── docker.go
│   ├── git.go              Use: gst
│   ├── jars.go
│   └── version.go
├── internal/
│   ├── tui/                Bubble Tea：各命令 UI + widgets/styles
│   ├── maven/              发现项目、跑 mvn
│   ├── jars/               jar 文件操作
│   ├── deploy/             deploy.yaml
│   ├── upload/             servers.yaml、SFTP、串行重启
│   ├── docker/             远程 docker 列表与跟日志重启
│   └── git/                多仓 status + 变更树
└── .wiki/                  本 Wiki
```

## 有意不存在的内容

- 无 `README.md`（仓库根）
- 无 `vendor/`（[gitignore](../../.gitignore) 忽略）
- 无 CI、无 Dockerfile
- `.idea/` 被 gitignore，不是运行时的一部分

## TUI 文件对照

| 文件 | 命令 |
| --- | --- |
| [internal/tui/mvn.go](../../internal/tui/mvn.go) | `mvn` |
| [internal/tui/upload.go](../../internal/tui/upload.go) | `deploy` |
| [internal/tui/docker.go](../../internal/tui/docker.go) | `docker` |
| [internal/tui/git.go](../../internal/tui/git.go) | `gst` |
| [internal/tui/jars.go](../../internal/tui/jars.go) | `jars` |
| [internal/tui/widgets.go](../../internal/tui/widgets.go) / [styles.go](../../internal/tui/styles.go) | 进度条、对齐、样式 |

## Related

- Code: [main.go](../../main.go)
- Modules: [../01-overview/module-map.md](../01-overview/module-map.md)
- Dependencies: [module-dependencies.md](module-dependencies.md)
