## 1. 添加 banner 文件支持

- [x] 1.1 在 `internal/context/loader.go` 的 CreateDefaultFiles 中添加 banner.txt
- [x] 1.2 在 `internal/context/loader.go` 中添加 LoadBanner 函数
- [x] 1.3 在 `cmd/autocode/main.go` 中读取并显示 banner

## 2. 测试

- [x] 2.1 Build: `go build ./cmd/autocode`
- [x] 2.2 Run: `./autocode` - 验证默认 banner 显示
- [x] 2.3 验证 banner.txt 文件已创建在 ~/.config/autocode/
- [ ] 2.4 自定义测试：修改 banner.txt 内容，验证显示更新
- [ ] 2.5 禁用测试：删除 banner.txt，验证无 banner 输出