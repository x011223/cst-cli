# 模块依赖

直接依赖以 `import` 为准（[go.mod](../../go.mod)）。

## 内部包

```text
cmd
 ├── tui
 └── docker          # 只取 DefaultLogTimeout

tui
 ├── maven
 ├── jars
 ├── deploy
 ├── upload
 ├── docker
 └── git

upload → jars
deploy → jars
```

领域包之间：`maven`、`git`、`docker` 互不 import。`upload` 与 `docker` 都依赖 `golang.org/x/crypto/ssh`。

## 主要第三方库

| 库 | 用途 |
| --- | --- |
| `github.com/spf13/cobra` | 子命令 |
| `github.com/charmbracelet/bubbletea` | TUI 程序 |
| `github.com/charmbracelet/lipgloss` | 颜色与宽度（`lipgloss.Width`） |
| `github.com/pkg/sftp` | SFTP 上传 |
| `golang.org/x/crypto/ssh` | SSH |
| `gopkg.in/yaml.v3` | 两份 YAML |

`go.mod` 里 `sftp` / `yaml` / `x/crypto` 标为 indirect，但源码直接 import；以源码为准。

## Related

- Code: [go.mod](../../go.mod)
- Architecture: [../01-overview/architecture.md](../01-overview/architecture.md)
