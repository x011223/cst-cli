# 编码约定

仓库没有独立 `CONTRIBUTING` / linter 配置。下列规则来自现有代码与已踩过的 TUI 坑。

## 包与入口

- 新命令：在 `cmd/` 加 `*Cmd`，于 [cmd/root.go](../../cmd/root.go) `AddCommand`；交互放 `internal/tui`，副作用放 `internal/<域>`
- 领域包不要 import `internal/tui`

## TUI 绘制

- 全屏命令使用 `tea.NewProgram(..., tea.WithAltScreen())`
- **不要把 `\n` 放进 lipgloss `Render()` 的字符串里**，否则后续行会对齐错位。多行用多次 `WriteString` + `\n`，单元格用 `padRight` / `tableRow`（[internal/tui/widgets.go](../../internal/tui/widgets.go)）
- 可见宽度用 `lipgloss.Width`，不要用 `len` 处理着色文本
- 过长日志用 `clipWidth`

## 并发与消息

- 后台 goroutine 只往 `chan tea.Msg` 发消息；改 model 只在 `Update`
- 并行日志发送使用带缓冲 channel，满则丢行（`docker` TUI `select/default`），避免阻塞 SSH 读循环

## 远程命令安全

- 容器名必须经过 `validName` / `validContainerName` 再拼进 `docker restart` / `logs`

## 配置

- 用户密钥只进 `~/.config/cst-cli/`，不要写进仓库或 Wiki

## Related

- Code: [internal/tui/widgets.go](../../internal/tui/widgets.go), [internal/tui/styles.go](../../internal/tui/styles.go)
- Testing: [testing.md](testing.md)
- Change guide: [../99-reference/change-guide.md](../99-reference/change-guide.md)
