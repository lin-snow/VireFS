# Contributing to VireFS

感谢你对 VireFS 的关注！无论是报告 Bug、提出建议还是贡献代码，我们都非常欢迎。

请先阅读我们的[行为准则](CODE_OF_CONDUCT.md)。

## 报告 Bug

请通过 [Bug 报告模板](https://github.com/lin-snow/VireFS/issues/new?template=bug_report.yml) 提交，模板会引导你填写 Go 版本、操作系统、复现步骤等关键信息。

## 提出功能建议

请通过 [功能建议模板](https://github.com/lin-snow/VireFS/issues/new?template=feature_request.yml) 描述你的使用场景和建议方案。

## 安全问题

请不要通过公开 Issue 报告安全漏洞，详见 [SECURITY.md](SECURITY.md)。

## 贡献代码

### 开发环境

- Go 1.25 或更高版本（与 `go.mod` 一致）
- Git

### 流程

1. Fork 本仓库
2. 创建功能分支：`git checkout -b feat/your-feature`
3. 编写代码并添加测试
4. 确保所有本地检查通过（与 CI 一致）：

```bash
go mod tidy            # 确保依赖干净
gofmt -l .             # 格式检查（无输出即通过）
go vet ./...           # 静态分析
go test -race ./...    # 竞态检测测试
```

5. 提交更改（请遵循下方 Commit 规范）
6. 推送到你的 Fork：`git push origin feat/your-feature`
7. 创建 Pull Request（仓库提供了 PR 模板，请按模板填写）

### CI 流水线

每个 PR 和 main 分支的推送会自动触发 [CI](.github/workflows/ci.yml)，包括：

- `go mod tidy` 一致性校验
- `gofmt` 格式检查
- `go vet` 静态分析
- `go test -race` 竞态测试

所有检查必须通过后才可合并。

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

- 按 PR 模板填写变更说明
- 关联相关 Issue（如有）
- 所有 CI 检查通过
- 新功能需附带测试

## 版本发布

维护者在 main 分支上打 `vX.Y.Z` 格式的 tag 后，[Release 工作流](.github/workflows/release.yml) 会自动运行测试并创建 GitHub Release（附自动生成的 Release Notes）。

## 许可

提交代码即表示你同意将贡献以 [Apache License 2.0](LICENSE) 许可发布。
