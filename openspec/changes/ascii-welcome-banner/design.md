## Context

Banner 在程序启动时显示，存储在配置目录中，用户可自定义。

特点：
1. **可自定义** - 用户可编辑 `~/.config/autocode/banner.txt`
2. **自动创建** - 首次运行时自动创建默认 banner 文件
3. **简洁** - 无额外依赖，直接读取文件显示

## Design

### Banner 风格

默认使用 figlet "shadow" 风格，存储在配置目录：

```
    #                         #####
   # #   #    # #####  ####  #     #  ####  #####  ######
  #   #  #    #   #   #    # #       #    # #    # #
 #     # #    #   #   #    # #       #    # #    # #####
 ####### #    #   #   #    # #       #    # #    # #
 #     # #    #   #   #    # #     # #    # #    # #
 #     #  ####    #    ####   #####   ####  #####  ######
```

特点：
- **可自定义** - 用户可编辑 `~/.config/autocode/banner.txt`
- **默认风格** - figlet shadow 风格，使用 `#` 符号
- **宽度约60字符** - 适合80列终端
- **自动创建** - 首次运行时自动创建默认 banner 文件

### Implementation

**1. 在 `internal/context/loader.go` 中添加默认 banner**

```go
// CreateDefaultFiles 中添加
"banner.txt": `...ASCII art content...`
```

**2. 在 `internal/context/loader.go` 中添加 LoadBanner 函数**

```go
// LoadBanner reads banner.txt for startup display
func (cl *ContextLoader) LoadBanner() string {
    path := filepath.Join(cl.configDir, "banner.txt")
    data, err := os.ReadFile(path)
    if err != nil {
        return ""
    }
    return string(data)
}
```

**3. 在 `cmd/autocode/main.go` 中读取并显示**

```go
// Display welcome banner (after CreateDefaultFiles)
loader := appctx.NewContextLoader(configPath, ".")
banner := loader.LoadBanner()
if banner != "" {
    fmt.Print(banner)
}
```

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| Banner 文件不存在 | LoadBanner 返回空字符串，不显示任何内容 |
| Banner 文件损坏/乱码 | 直接输出文件内容，用户可自行修复 |
| 用户想禁用 banner | 删除或清空 banner.txt 文件即可 |

## Testing

1. 首次运行：`./autocode` - 验证默认 banner 创建并显示
2. 修改 banner：编辑 `~/.config/autocode/banner.txt`，验证自定义内容显示
3. 禁用 banner：删除 banner.txt，验证无输出
4. 宽度测试：在小终端中运行，验证不换行