# 简介

AutoCode 是一个全自动的 Harness Engineering 框架。它支持：
- 接到任务时，进入规划模式，先划分成若干个可独立运行、测试验证的模块；
- 每个模块都有需求文档、编码、验证、测试的完整流程；
- 把每个模块拆解成一个个可验证的任务，每个做完一个任务，提交到git仓库，写清楚本次的提交；
- 每个任务完成后，必须做代码走查，及时修复严重问题；暂时搁置的问题，及时记录到Github Issue后续跟进；
- 每个任务除了跑本地的单元测试，也需要部署到测试服务器，验证切实可行。
- 做完每个模块后，Review模块相关的GitHub Issue，尽可能解决；

我希望你从现在开始，也用如上规则要求自己。

# 环境

仓库： github.com/zend/AutoCode
使用 `gh` 命令管理github仓库。

测试机器： claw@10.10.1.5 你可以使用sudo安装需要的软件。
尽可能保持机器的干净、安全，临时性的工作尽量使用 docker，用完清理。

# 构建命令

```bash
# 构建
go build -o bin/autocode ./cmd/autocode

# 运行
./bin/autocode

# 运行所有测试
go test ./...

# 运行单个包的测试
go test ./internal/llm/...

# 运行单个测试函数
go test -run TestChat ./internal/llm/...

# 运行测试并显示覆盖率
go test -cover ./...

# 生成覆盖率报告
go test -coverprofile=coverage.txt ./... && go tool cover -html=coverage.txt

# 代码格式化
go fmt ./...

# 静态检查（如果安装了 golangci-lint）
golangci-lint run

# 或者使用 go vet
go vet ./...
```

# 代码风格指南

## 导入

```go
import (
    // 标准库
    "context"
    "fmt"
    
    // 第三方库
    tea "github.com/charmbracelet/bubbletea"
    
    // 本地包
    "github.com/zend/AutoCode/internal/llm"
)
```

- 按：标准库 → 第三方库 → 本地包 分组，组间空行
- 第三方库可使用别名简化（如 `tea "github.com/charmbracelet/bubbletea"`）

## 命名约定

- **包名**：小写单词，不使用下划线（如 `llm`）
- **类型**：导出类型使用 PascalCase，注释描述用途（如 `// Client is the LLM client`）
- **接口**：单方法接口以 "-er" 结尾（如 `Reader`, `Writer`）
- **常量**：PascalCase 或全大写+下划线
- **私有字段**：camelCase（如 `baseURL`, `apiKey`）
- **测试函数**：`Test<功能名>`（如 `TestChat`, `TestNewClient`）

## 错误处理

```go
// 使用 fmt.Errorf 包装错误，保留调用链
if err != nil {
    return nil, fmt.Errorf("marshal request: %w", err)
}

// API 错误应包含状态码和响应体
if resp.StatusCode != http.StatusOK {
    return nil, fmt.Errorf("api error: %s - %s", resp.Status, string(respBody))
}
```

- 不忽略错误，必须处理
- 使用 `%w` 包装错误以支持 `errors.Is`/`errors.As`
- 错误信息以小写开头，不标点结尾

## 类型定义

```go
// 结构体字段使用 json tag，可选字段用 omitempty
type ChatRequest struct {
    Model       string    `json:"model"`
    Messages    []Message `json:"messages"`
    Temperature float64   `json:"temperature,omitempty"`
    MaxTokens   int       `json:"max_tokens,omitempty"`
}
```

- 导出类型必须有注释
- JSON tag 使用 snake_case
- 可选字段添加 `omitempty`

## 测试规范

```go
func TestChat(t *testing.T) {
    // 使用 httptest 创建模拟服务器
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 验证请求
        if r.URL.Path != "/chat/completions" {
            t.Errorf("expected path /chat/completions, got %s", r.URL.Path)
        }
        // 返回响应
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{...}`))
    }))
    defer server.Close()
    
    // 测试逻辑
    client := NewClient(server.URL, "test-key")
    resp, err := client.Chat(context.Background(), ChatRequest{...})
    if err != nil {
        t.Fatalf("Chat failed: %v", err)
    }
    // 断言
}
```

- 使用 `testing` 包
- 表驱动测试用于多场景测试
- 使用 `httptest` 模拟 HTTP 服务
- 失败时使用 `t.Fatalf` 停止测试，`t.Errorf` 继续测试

## 上下文使用

```go
// 所有阻塞操作应支持 context
func (c *Client) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
    httpReq, err := http.NewRequestWithContext(ctx, "POST", url, body)
    // ...
}
```

- 使用 `context.Context` 作为第一个参数
- 使用 `http.NewRequestWithContext` 支持超时和取消

## 代码组织

```
AutoCode/
├── cmd/                    # 主程序入口
│   └── autocode/
│       └── main.go
├── internal/               # 内部包（不对外暴露）
│   └── llm/
│       ├── client.go
│       └── client_test.go
├── go.mod
├── go.sum
└── README.md
```

- `cmd/` 存放可执行程序入口
- `internal/` 存放内部实现，每个子目录一个包
- 测试文件与源文件同目录，命名为 `<name>_test.go`

# 具体工作内容

- ReAct Agent 主流程，只需要支持 OpenAI / Anthropic 这两种兼容端点，不使用官方直连
- 工具集（需要你实现）：
    * Read: 
        + 支持读目录结构，类似tree，但尊重 .gitignore，忽略编译产生的文件；
        + 支持读文件，最多100行，带行号返回，支持传入起始行号
        + 理解文件时间
    * Write:
        + 精准匹配替换，修改前文件时间校验，代码块原文校验，预期替换几处数量校验
        + 改完先lint无误才成功，拒绝交付通不过lint的代码
    * Grep:
        + 高级grep，尊重 .gitignore，忽略编译产生的文件；
        + 单行过长的匹配项，直接过滤（大概率是压缩js/css）
        + 忽略二进制匹配
    * Shell

# 注意事项

- 提交前必须运行 `go fmt ./...` 和 `go vet ./...`
- 所有新功能必须包含单元测试
- 保持代码简洁，避免过度抽象
- 使用 Go 1.24+ 特性（如 toolchain 指令）