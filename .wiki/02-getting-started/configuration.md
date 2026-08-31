# 配置

两份本机 YAML，默认都在 `~/.config/cst-cli/`。仓库不包含真实配置；下面字段名与默认值来自源码。**不要把密码或真实主机写进 Wiki / git。**

## servers.yaml

加载：[internal/upload/upload.go](../../internal/upload/upload.go) `LoadConfig`、`DefaultConfigPath`。

| 字段 | 含义 | 默认 |
| --- | --- | --- |
| `environments` | 环境列表，不能为空 | — |
| `name` | TUI / `-e` 使用的环境名 | — |
| `host` | SSH 主机 | — |
| `port` | SSH 端口 | `22` |
| `user` | SSH 用户 | — |
| `password` | 密码（同时用于 keyboard-interactive） | — |
| `destDir` | SFTP 上传目录 | `/tmp` |

占位示例（值请换成自己的，勿提交）：

```yaml
environments:
  - name: example
    host: 127.0.0.1
    port: 22
    user: deploy
    password: "changeme"
    destDir: /data/cst/app/jar
```

`deploy` 与 `docker` 共用这份文件。

## deploy.yaml

加载：[internal/deploy/config.go](../../internal/deploy/config.go) `LoadConfig`。`services` 不能为空。

| 字段 | 含义 | 默认 |
| --- | --- | --- |
| `localJarDir` | 本机暂存目录（`mvn` 写入、`deploy` 读取） | `~/Documents/Jars` |
| `jarDir` | 远程已部署 jar 目录（解析后写入 Config；当前上传实际用 `servers.yaml` 的 `destDir`） | `/data/cst/app/jar` |
| `tmpDir` | 注释写明 unused leftover，兼容旧文件 | `/tmp` |
| `services[].name` | 展示名 | — |
| `services[].jar` | 精确 jar 文件名（`mvn` 拷贝、`deploy` 映射） | — |
| `services[].container` | 远程 Docker 容器名 | — |

`MatchServices` 按 jar 名匹配，跳过重复 container，未匹配的 jar 进入 `unmatched`。

占位示例：

```yaml
localJarDir: ~/Documents/Jars
services:
  - name: system
    jar: system-application-2.0.0.jar
    container: commsoft-system
  - name: auth
    jar: commsoft-auth.jar
    container: commsoft-auth
```

测试里的映射样例见 [internal/deploy/config_test.go](../../internal/deploy/config_test.go)。

## 环境变量

源码中**没有**读取 `CST_*` 一类业务环境变量。Go 模块代理等属于本机 Go 工具链，不是本项目配置。

## Related

- Code: [internal/upload/upload.go](../../internal/upload/upload.go), [internal/deploy/config.go](../../internal/deploy/config.go)
- Flows: [../04-business/core-flows.md](../04-business/core-flows.md)
- Integrations: [../08-external-integrations/integrations.md](../08-external-integrations/integrations.md)
