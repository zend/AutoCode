## Why

AutoCode 目前启动时没有任何视觉标识。添加一个欢迎 banner 可以：

- 提供即时的视觉反馈，确认程序已启动
- 建立品牌识别
- 用户可自定义 banner 内容

## What Changes

- 在配置目录创建默认 `banner.txt` 文件
- 启动时读取并显示 banner
- 用户可编辑 `~/.config/autocode/banner.txt` 自定义内容
- 删除文件可禁用 banner

## Capabilities

### New Capabilities
- `welcome-banner`: 启动时显示 ASCII art banner

## Impact

- `internal/context/loader.go`: 添加 banner.txt 默认文件和 LoadBanner 函数
- `cmd/autocode/main.go`: 加载并显示 banner