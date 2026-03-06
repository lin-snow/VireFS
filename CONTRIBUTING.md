# Contributing to VireFS

感谢你对 VireFS 的关注！无论是报告 Bug、提出建议还是贡献代码，我们都非常欢迎。

## 报告 Bug

请通过 [GitHub Issues](https://github.com/lin-snow/VireFS/issues/new?template=bug_report.yml) 提交 Bug 报告，并尽量包含：

- Go 版本和操作系统
- 最小可复现代码
- 期望行为与实际行为

## 提出功能建议

请通过 [GitHub Issues](https://github.com/lin-snow/VireFS/issues/new?template=feature_request.yml) 描述你的使用场景和建议方案。

## 贡献代码

### 开发环境

- Go 1.24 或更高版本
- Git

### 流程

1. Fork 本仓库
2. 创建功能分支：`git checkout -b feat/your-feature`
3. 编写代码并添加测试
4. 确保所有检查通过：

```bash
gofmt -l .
go vet ./...
go test -race ./...
```

5. 提交更改（请遵循下方 Commit 规范）
6. 推送到你的 Fork：`git push origin feat/your-feature`
7. 创建 Pull Request

### Commit 规范

使用 [Conventional Commits](https://www.conventionalcommits.org/) 格式：

```
<type>: <description>

[optional body]
```

常见 type：

| Type | 说明 |
|------|------|
| `feat` | 新功能 |
| `fix` | Bug 修复 |
| `docs` | 文档变更 |
| `test` | 测试相关 |
| `refactor` | 重构（不改变功能） |
| `chore` | 构建/工具变更 |

示例：`feat: add TTL support for ObjectFS`

### 代码风格

- 使用 `gofmt` 格式化代码
- 通过 `go vet` 静态检查
- 导出的类型和函数需要 GoDoc 注释
- 表驱动测试优先

### Pull Request 要求

- 描述清楚变更内容和原因
- 关联相关 Issue（如有）
- 所有 CI 检查通过
- 新功能需附带测试

## 许可

提交代码即表示你同意将贡献以 [Apache License 2.0](LICENSE) 许可发布。
