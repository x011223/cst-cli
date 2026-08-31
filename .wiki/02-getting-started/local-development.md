# 本机开发与运行

从仓库根目录构建并调用子命令。

## 构建

```bash
go build -o cst-cli .
```

覆盖版本（可选）：

```bash
go build -ldflags "-X github.com/wujunqiang/cst-cli/cmd.Version=x.y.z" -o cst-cli .
```

入口与版本定义：[main.go](../../main.go)、[cmd/root.go](../../cmd/root.go)、[cmd/version.go](../../cmd/version.go)。

## 测试

```bash
go test ./...
```

有测试的包：`internal/maven`、`internal/jars`、`internal/deploy`、`internal/upload`、`internal/docker`、`internal/tui`（widgets）。`cmd` 无测试文件。

## 常用命令

在**含多个 Maven 子项目**的目录执行 `mvn` / `jars`；在任意目录执行 `gst`；`deploy` / `docker` 依赖本机 YAML。

```bash
./cst-cli version
./cst-cli mvn -e dev
./cst-cli deploy -e <env-name>
./cst-cli docker -e <env-name>
./cst-cli gst
./cst-cli jars -d ~/Documents/Jars/
```

| Flag | 命令 | 含义 |
| --- | --- | --- |
| `-e` / `--env` | `mvn` | Maven profile：`dev` / `test` / `prod`（写死在 [internal/tui/mvn.go](../../internal/tui/mvn.go) `envOptions`） |
| `-e` / `--env` | `deploy`, `docker` | `servers.yaml` 里的 `environments[].name`，跳过环境选择屏 |
| `-c` / `--config` | `deploy`, `docker` | `servers.yaml` 路径，默认 `~/.config/cst-cli/servers.yaml` |
| `--deploy-config` | `deploy` | `deploy.yaml` 路径 |
| `-p` / `--pattern` | `deploy`, `jars` | 逗号分隔的 jar 名 glob |
| `-d` / `--dest` | `jars` | 拷贝目标，默认 `~/Documents/Jars/` |
| `--timeout` | `docker` | 等待日志含「启动成功」的时长，默认 2 分钟 |

## Related

- Code: [cmd/](../../cmd/)
- Prerequisites: [prerequisites.md](prerequisites.md)
- Commands: [../04-business/command-domains.md](../04-business/command-domains.md)
