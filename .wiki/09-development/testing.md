# 测试

全部为 `testing` 包单测，无集成测试、无远程 SSH mock 套件。

## 如何跑

```bash
go test ./...
```

需要拉模块时，以本机 `GOPROXY` 为准（项目本身不设置）。

## 覆盖范围

| 包 | 文件 | 覆盖什么 |
| --- | --- | --- |
| `internal/maven` | [build_test.go](../../internal/maven/build_test.go) | `FindMavenProjects` 扫描；`RunBuilds` 会对临时目录真调 `mvn`（假 pom 预期失败） |
| `internal/jars` | [jars_test.go](../../internal/jars/jars_test.go) | 过滤、目录、ClearDir 安全边界 |
| `internal/deploy` | [config_test.go](../../internal/deploy/config_test.go) | `JarNames`、`MatchServices`、`ResolveLocalJarDir` |
| `internal/upload` | [upload_test.go](../../internal/upload/upload_test.go) | 容器名校验、`progressReader` 字节回调 |
| `internal/docker` | [docker_test.go](../../internal/docker/docker_test.go) | `ParsePS`、`IsReady`、名称校验 |
| `internal/tui` | [widgets_test.go](../../internal/tui/widgets_test.go) | `FormatSize`、进度条、`padRight`、重启记录对齐、`annotateJarProjects` |

无测试：`cmd/*`、各 `Run*` TUI 状态机、真实 SSH/SFTP。

## Related

- Code: 上表路径
- Local run: [../02-getting-started/local-development.md](../02-getting-started/local-development.md)
