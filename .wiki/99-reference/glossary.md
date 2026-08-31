# 术语

| 术语 | 含义（本仓库） |
| --- | --- |
| 暂存目录 / staging / `localJarDir` | `deploy.yaml` 的本机 jar 目录，默认 `~/Documents/Jars` |
| servers.yaml | SSH 环境列表；`deploy` 与 `docker` 共用 |
| `destDir` | `servers.yaml` 里该环境的 SFTP 上传目录；不同环境可以不同 |
| deploy.yaml | 本机 `localJarDir` + jar 精确文件名 ↔ 容器名；不含远程上传路径 |
| `SuccessMarker` | 字符串 `启动成功`，`docker` 命令认为应用已起来 |
| `RestartPause` | `deploy` 串行重启间隔，5 秒 |
| `DefaultLogTimeout` | `docker` 跟日志超时，2 分钟 |
| gst | git 状态子命令的 `Use` 名 |
| deloy | `deploy` 的拼写别名 |
| application jar | `jars` 默认 glob `*-application*.jar`；`mvn` 暂存改为 YAML 里的精确名 |

## Related

- Config: [../02-getting-started/configuration.md](../02-getting-started/configuration.md)
- Commands: [../04-business/command-domains.md](../04-business/command-domains.md)
