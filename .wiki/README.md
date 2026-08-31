# cst-cli Wiki

本目录是 **cst-cli** 的项目 Wiki：面向开发人员与 AI 的导航，内容均来自当前仓库源码与配置，不含密钥与真实主机地址。

仓库根目录没有 `README.md`。日常用法以本 Wiki 的 [快速开始](02-getting-started/local-development.md) 与 [命令域](04-business/command-domains.md) 为准。

## 阅读路径

1. [项目是什么](01-overview/project-overview.md) → [架构](01-overview/architecture.md) → [模块地图](01-overview/module-map.md)
2. [本机运行](02-getting-started/local-development.md) → [配置文件](02-getting-started/configuration.md)
3. 改代码：[仓库结构](03-codebase/repository-structure.md) → [核心流程](04-business/core-flows.md) → [约定](09-development/coding-conventions.md)

## 目录

| 分区 | 内容 |
| --- | --- |
| [01-overview](01-overview/project-overview.md) | 产品定位、架构、模块职责 |
| [02-getting-started](02-getting-started/prerequisites.md) | 依赖、构建、YAML 配置 |
| [03-codebase](03-codebase/repository-structure.md) | 目录与包依赖 |
| [04-business](04-business/command-domains.md) | 子命令与端到端流程 |
| [08-external-integrations](08-external-integrations/integrations.md) | Maven / git / SSH / Docker |
| [09-development](09-development/coding-conventions.md) | TUI 约定与测试 |
| [99-reference](99-reference/glossary.md) | 术语与改代码入口 |

未设立 `05-api/`、`06-data/`、`10-deployment/`：本仓库不是 HTTP 服务，没有数据库，也没有独立的 CLI 发布流水线。

## Related Documents

- 源码入口：[main.go](../main.go)
- 命令注册：[cmd/root.go](../cmd/root.go)
- 依赖声明：[go.mod](../go.mod)
