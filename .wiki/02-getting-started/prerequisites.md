# 前置条件

运行与开发 cst-cli 所需的本机工具。配置文件本身见 [configuration.md](configuration.md)。

## 构建与运行 CLI

| 依赖 | 用途 | 证据 |
| --- | --- | --- |
| Go（[go.mod](../../go.mod) 写明 `go 1.26.6`） | `go build` / `go test` | 模块文件 |
| 终端（建议支持 alt screen） | Bubble Tea 全屏 UI | `tea.WithAltScreen()` |

仓库无 `Makefile`、无 Dockerfile、无 CI 配置。

## 子命令额外依赖

| 工具 | 谁调用 | 不满足时 |
| --- | --- | --- |
| `mvn` 在 `PATH` | [internal/maven/build.go](../../internal/maven/build.go) `exec.Command("mvn", …)` | 构建失败，TUI 展示 Maven 输出摘要 |
| `git` 在 `PATH` | [internal/git/status.go](../../internal/git/status.go) `exec.Command("git", …)` | 该仓库被跳过或发现失败 |
| 远程主机 `docker` CLI | `upload.RestartContainers`、`docker.List` / `RestartAndFollow` | SSH 命令失败，界面标失败 |
| 本机 YAML | `~/.config/cst-cli/servers.yaml`（deploy/docker）、`deploy.yaml`（mvn/deploy） | `LoadConfig` 返回错误 |

`gst` / `jars` / `version` 不要求 `servers.yaml`。

## 工作目录约定

- `mvn` / `jars`：当前目录的**一层子目录**中含 `pom.xml` 才算 Maven 项目（[internal/maven/detect.go](../../internal/maven/detect.go)）。当前目录自己有 `pom.xml` 但子目录没有时，列表为空。
- `gst`：当前目录或一层子目录存在 `.git` 即视为仓库（[internal/git/status.go](../../internal/git/status.go) `Discover`）。

## Related

- Code: [go.mod](../../go.mod), [internal/maven/build.go](../../internal/maven/build.go)
- Local run: [local-development.md](local-development.md)
- Config: [configuration.md](configuration.md)
