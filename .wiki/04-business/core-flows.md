# 核心流程

下面调用链均可在源码中对上。没有前端或 HTTP 层。

## 1. 构建并暂存 jar

```text
cst-cli mvn [-e dev]
  → cmd/mvn.go RunE
  → tui.RunMvnBuild
       → maven.FindMavenProjects(cwd)          # 仅一层子目录 + pom.xml
       → deploy.LoadConfig("")                 # 可选失败，仍可构建
       → TUI: 选 env / 选项目
       → jars.ClearDir(localJarDir)
       → maven.RunBuilds(projects, profile)
            → exec mvn -B [-Pprofile] clean|compile|package
       → stageBuiltJars
            → jars.FindJars(成功项目)
            → jars.FilterExact(..., deployCfg.JarNames())
            → jars.CopyJars(..., localJarDir)
```

## 2. 上传并串行重启（deploy）

```text
cst-cli deploy [-e name]
  → tui.RunDeploy
       → upload.LoadConfig(servers.yaml)
       → deploy.LoadConfig(deploy.yaml)
       → jars.ListDir(localJarDir)
       → TUI: 选环境 / 选 jar
       → upload.UploadAll(env, files)
            → 每文件 Environment.dial + sftp.Create(destDir/name)
       → afterUpload: deploy.MatchServices(成功 jar 名)
       → 确认后 upload.RestartContainers(..., RestartPause=5s)
            → 单 SSH：依次 docker restart <container>
  → 若 uploadedOK：jars.ClearDir(localJarDir)
```

## 3. 分组并行重启并跟日志（docker）

```text
cst-cli docker [-e name]
  → tui.RunDocker
       → upload.LoadConfig + Environment.Dial
       → docker.List → docker ps -a --format ...
       → 多选一组 → startGroup
            → 每容器 goroutine: docker.RestartAndFollow
                 → docker restart <name>
                 → docker logs -f --tail 80 <name>
                 → 行含「启动成功」则成功
       → 全部结束后回列表并 refresh，marks 按容器名保留
```

## 4. 多仓 git 变更

```text
cst-cli gst
  → tui.RunGitStatus
       → git.Discover(cwd)
            → git rev-parse --abbrev-ref HEAD
            → git status --porcelain=v1 -uall
       → RepoStatus.Tree() 绘制
```

## 5. 按 glob 收集 jar

```text
cst-cli jars
  → tui.RunJars
       → maven.FindMavenProjects + jars.FindJars
       → jars.FilterByName(DefaultJarPattern 或 -p)
       → jars.CopyJars(选中项, dest)
```

## Related

- Code: [internal/tui/mvn.go](../../internal/tui/mvn.go), [internal/tui/upload.go](../../internal/tui/upload.go), [internal/tui/docker.go](../../internal/tui/docker.go)
- Commands: [command-domains.md](command-domains.md)
- Integrations: [../08-external-integrations/integrations.md](../08-external-integrations/integrations.md)
