# 外部集成

本仓库没有 HTTP client SDK。对外能力是本机进程与 SSH 上的远程 CLI。

## Apache Maven

- 调用：`mvn -B [-P<profile>] <phase>`，工作目录为项目路径（[internal/maven/build.go](../../internal/maven/build.go) `runMaven`）
- profile 来自 TUI `envOptions` 或 `-e`，不是 `servers.yaml`
- 失败时把 stdout+stderr 带回 TUI，`mavenErrorSummary` 抽 `[ERROR]` / `BUILD FAILURE`

## Git

- `rev-parse --abbrev-ref HEAD` 与 `status --porcelain=v1 -uall`（[internal/git/status.go](../../internal/git/status.go)）
- 只扫 cwd 与一层子目录；嵌套更深的仓库不会单独列出

## SSH / SFTP

- 实现：[internal/upload/upload.go](../../internal/upload/upload.go) `dial`
- 认证：密码 + keyboard-interactive（问题一律回填 password）
- 主机密钥：`ssh.InsecureIgnoreHostKey()`（源码注释：trusted networks）
- 超时：15s
- 上传：`github.com/pkg/sftp`，`MkdirAll(destDir)` 后按文件名写入
- `docker` 命令通过公开方法 `Environment.Dial()` 复用同一 dial

## 远程 Docker

| 能力 | 实现 | 命令 |
| --- | --- | --- |
| 列表 | [internal/docker/docker.go](../../internal/docker/docker.go) `List` | `docker ps -a --format '{{.Names}}\|{{.State}}\|{{.Status}}\|{{.Image}}'` |
| deploy 串行重启 | `upload.restartOne` | `docker restart <name>` |
| docker 跟日志重启 | `RestartAndFollow` | `docker restart` 然后 `docker logs -f --tail 80` |
| 就绪 | `IsReady` | 日志子串 `启动成功` |

容器名校验：字母数字 `_` `-` `.`，以及非首位的 `:`（[validName](../../internal/docker/docker.go) / `validContainerName`）。拒绝空格与 shell 元字符，避免拼进 SSH 命令。

## 本机文件系统

- 暂存默认 `~/Documents/Jars`（`jars.ExpandHome`）
- `ClearDir` 拒绝空路径、`/`、用户 home

## Related

- Code: [internal/upload/upload.go](../../internal/upload/upload.go), [internal/docker/docker.go](../../internal/docker/docker.go)
- Config: [../02-getting-started/configuration.md](../02-getting-started/configuration.md)
- Architecture: [../01-overview/architecture.md](../01-overview/architecture.md)
