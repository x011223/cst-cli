# 模块地图

按 Go 包列出职责、入口与依赖方向。包名以源码 `package` 为准。

| 包 | 路径 | 职责 | 对外入口 |
| --- | --- | --- | --- |
| `main` | [main.go](../../main.go) | 进程入口 | `main()` |
| `cmd` | [cmd/](../../cmd/) | 注册子命令与 flags | `Execute()` |
| `tui` | [internal/tui/](../../internal/tui/) | 全部交互界面 | `RunMvnBuild` / `RunDeploy` / `RunDocker` / `RunGitStatus` / `RunJars` |
| `maven` | [internal/maven/](../../internal/maven/) | 扫描子目录 `pom.xml`；并行构建、项目内串行 phase | `FindMavenProjects`, `RunBuilds` |
| `jars` | [internal/jars/](../../internal/jars/) | 找 jar、过滤、拷贝、列暂存目录、安全清空 | `FindJars`, `ListDir`, `ClearDir`, `FilterExact` |
| `deploy` | [internal/deploy/](../../internal/deploy/) | 加载 `deploy.yaml`，jar↔container 映射 | `LoadConfig`, `MatchServices`, `JarNames` |
| `upload` | [internal/upload/](../../internal/upload/) | 加载 `servers.yaml`，SFTP 上传，串行重启 | `LoadConfig`, `UploadAll`, `RestartContainers`, `Environment.Dial` |
| `docker` | [internal/docker/](../../internal/docker/) | 远程 `docker ps` / `restart` / `logs -f`，成功判定 | `List`, `RestartAndFollow`, `IsReady` |
| `git` | [internal/git/](../../internal/git/) | 一层 git 仓库发现与变更树 | `Discover`, `RepoStatus.Tree` |

## 依赖方向

```text
cmd → tui
cmd → docker          （仅 docker 子命令读 DefaultLogTimeout）
tui → maven, jars, deploy, upload, docker, git
upload → jars         （ExpandHome / JarFile）
deploy → jars         （ExpandHome）
maven ↛ tui
docker ↛ tui
```

`internal/docker` 与 `internal/upload` 都建 SSH session；`docker` TUI 通过 `upload.Environment.Dial()` 复用同一套认证。

## Related

- Code: [cmd/root.go](../../cmd/root.go)
- Structure: [../03-codebase/repository-structure.md](../03-codebase/repository-structure.md)
- Dependencies: [../03-codebase/module-dependencies.md](../03-codebase/module-dependencies.md)
