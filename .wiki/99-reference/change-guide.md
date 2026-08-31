# 改代码入口

按常见改动找文件，避免在错误层加逻辑。

| 想改什么 | 先看 |
| --- | --- |
| 增加子命令 | [cmd/](../../cmd/) + [cmd/root.go](../../cmd/root.go) `AddCommand` |
| Maven 扫描范围 / phase | [internal/maven/detect.go](../../internal/maven/detect.go), [build.go](../../internal/maven/build.go) |
| 构建可选环境列表 | [internal/tui/mvn.go](../../internal/tui/mvn.go) `envOptions` |
| 暂存拷贝规则 | `stageBuiltJars`（[internal/tui/mvn.go](../../internal/tui/mvn.go)）+ [internal/jars/jars.go](../../internal/jars/jars.go) |
| 上传进度 / 并行上传 | [internal/upload/upload.go](../../internal/upload/upload.go) `UploadAll` |
| deploy 重启间隔 / 串行策略 | `RestartPause`、`RestartContainers` |
| docker 成功文案 / 超时 | [internal/docker/docker.go](../../internal/docker/docker.go) `SuccessMarker`, `DefaultLogTimeout` |
| docker 分组 UI / 分栏日志 | [internal/tui/docker.go](../../internal/tui/docker.go) |
| YAML 字段 | [internal/deploy/config.go](../../internal/deploy/config.go), [internal/upload/upload.go](../../internal/upload/upload.go) `Environment` |
| 进度条、对齐、截断 | [internal/tui/widgets.go](../../internal/tui/widgets.go) |
| 颜色 | [internal/tui/styles.go](../../internal/tui/styles.go) |

## 不要做的事

- 不要在 Wiki 或提交里写入 `servers.yaml` 密码
- 不要把多行字符串丢进单个 lipgloss `Render()`
- 不要把未校验的容器名拼进远程 shell

## Related

- Conventions: [../09-development/coding-conventions.md](../09-development/coding-conventions.md)
- Flows: [../04-business/core-flows.md](../04-business/core-flows.md)
