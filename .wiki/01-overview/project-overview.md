# 项目概览

cst-cli 是一个本地交互式开发者 CLI：把 Maven 构建、jar 暂存、SFTP 上传、远程 Docker 重启、多仓库 git 变更浏览收成一组 Cobra 子命令，界面用 Bubble Tea。

## 事实要点

- 模块路径：`github.com/wujunqiang/cst-cli`（[go.mod](../../go.mod)）
- Go 版本：`go 1.26.6`
- 版本字符串：`cmd.Version`，默认 `0.1.0`，可用 `-ldflags` 覆盖（[cmd/root.go](../../cmd/root.go)）
- 进程入口：`main()` → `cmd.Execute()`（[main.go](../../main.go)）
- 无前端、无 HTTP API、无本机数据库
- 用户配置在 `~/.config/cst-cli/`（`servers.yaml`、`deploy.yaml`），不在仓库内

## 子命令

| 命令 | 别名 | 入口 |
| --- | --- | --- |
| `mvn` | `maven` | [cmd/mvn.go](../../cmd/mvn.go) → `tui.RunMvnBuild` |
| `gst` | `git`, `status` | [cmd/git.go](../../cmd/git.go) → `tui.RunGitStatus` |
| `deploy` | `deloy`, `upload` | [cmd/deploy.go](../../cmd/deploy.go) → `tui.RunDeploy` |
| `docker` | `ps`, `containers` | [cmd/docker.go](../../cmd/docker.go) → `tui.RunDocker` |
| `jars` | — | [cmd/jars.go](../../cmd/jars.go) → `tui.RunJars` |
| `version` | `v` | [cmd/version.go](../../cmd/version.go) |

## 典型使用场景

在含多个 Maven 子目录的工作区：`mvn` 构建并按 `deploy.yaml` 的 jar 名拷到暂存目录 → `deploy` SFTP 上传并按映射串行重启容器；或单独用 `docker` 分组并行重启并跟日志。`gst` 看当前目录及一层子目录的 git 未提交变更。`jars` 是按 glob 收集 `target/*.jar` 的独立工具，不走 `deploy.yaml` 精确文件名。

## Related

- Code: [main.go](../../main.go), [cmd/root.go](../../cmd/root.go)
- Modules: [module-map.md](module-map.md)
- Configuration: [../02-getting-started/configuration.md](../02-getting-started/configuration.md)
- Flows: [../04-business/core-flows.md](../04-business/core-flows.md)
