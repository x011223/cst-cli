# 命令域

每个子命令是独立交互流程，共享暂存目录与 YAML 的只有 `mvn` ↔ `deploy`，以及 `deploy` / `docker` 共用 `servers.yaml`。

## mvn

- 选 profile（`dev` / `test` / `prod`）→ 多选一层子目录 Maven 项目
- Enter 前清空 `localJarDir`（[internal/tui/mvn.go](../../internal/tui/mvn.go)）
- 项目并行、每项目 `clean → compile → package` 串行，命令为 `mvn -B [-P<env>] <phase>`
- 成功项目用 `jars.FilterExact(..., cfg.JarNames())` 拷到暂存目录；`deploy.yaml` 缺失则不拷

## deploy（别名 deloy / upload）

- 读 `servers.yaml` + `deploy.yaml`，列出暂存目录中的 `.jar`
- 并行 SFTP 上传到环境 `destDir`
- 至少一个文件成功后，按 `MatchServices` 询问是否重启；确认后**串行** `docker restart`，每个完成后等 5s（`upload.RestartPause`）
- TUI 退出且至少一次上传成功时 `jars.ClearDir(localJarDir)`

## docker（别名 ps / containers）

- 同一套 SSH 连环境，`docker ps -a` 列表，多选一组
- 组内并行 `RestartAndFollow`：先 `docker restart`，再 `docker logs -f --tail 80`，直到日志含 `启动成功`（[internal/docker/docker.go](../../internal/docker/docker.go) `SuccessMarker`）、超时（默认 2m）或出错
- 每台容器单独日志面板；整组结束后回列表，已跑过的标 `✓ 已重启` / `✗ 已重启`

## gst（别名 git / status）

- `git.Discover(cwd)`：cwd 与一层子目录中带 `.git` 且有未提交变更的仓库
- 展示 branch 与变更树（[internal/git/tree.go](../../internal/git/tree.go)）

## jars

- 同样用 `FindMavenProjects` + `FindJars`，默认 glob `*-application*.jar`
- 交互勾选后拷到 `--dest`（默认 `~/Documents/Jars/`）
- **不**读 `deploy.yaml` 精确文件名，也**不**清空暂存（除非 dest 碰巧是同一目录且用户覆盖）

## version

打印 `cst-cli <Version>`。

## Related

- Code: [cmd/](../../cmd/)
- Flows: [core-flows.md](core-flows.md)
- Config: [../02-getting-started/configuration.md](../02-getting-started/configuration.md)
