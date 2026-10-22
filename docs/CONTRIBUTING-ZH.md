# LANdapter 贡献指南

**语言：** [English](CONTRIBUTING.md) · [Русский](CONTRIBUTING-RU.md) · [中文](CONTRIBUTING-ZH.md) · [日本語](CONTRIBUTING-JA.md) · [Español](CONTRIBUTING-ES.md)

欢迎为 LANdapter 提交错误报告、修复、文档更新和新功能。

## 提交 Issue 前

先搜索重复问题，并尽量在当前版本复现。请提供 Go、PostgreSQL、Node.js 和操作系统版本，复现步骤，预期与实际结果，以及必要日志和 UI 截图。上传日志前请删除 token、密码、个人信息和内部网络细节。

## 开发流程

```bash
git clone https://github.com/chocom1nt/LANdapter.git
cd LANdapter
git checkout -b fix/short-description
go mod download
cd web && npm ci && cd ..
```

建议分支名：`fix/...`、`feature/...`、`docs/...`。尽量让每个 PR 聚焦于一个问题，并说明修改原因。

## 测试与格式

```bash
gofmt -w ./cmd ./internal ./storage
go test ./...
make build
cd web && npm run build
```

如已安装 golangci-lint，可运行 `golangci-lint run ./...`。请使用 `go.mod` 指定的 Go 版本。不要声称测试或构建成功，除非实际执行并确认。

## 约定

- Go：使用 `gofmt`、显式处理错误，不在日志中记录密码/token，并为导出 API 添加注释。
- React：使用函数组件、清晰命名和明确的 API 错误处理。
- SQL：新增 `NNN_description.up.sql` 和对应 `.down.sql`；不要在不说明的情况下破坏性修改已发布的迁移。
- API/协议变更：同步更新 `README.API.md` 及俄语、中文、日语、西班牙语版本。
- 配置变更：更新 `CONFIG.md` 与各语言版本。
- 不要提交密钥、编译产物或 `node_modules`。

## Pull Request

说明问题、解决方案、执行过的测试、需要的迁移/配置以及兼容性风险。UI 改动请附不含真实数据的截图。安全相关改动应描述威胁模型，并尽可能加入回归测试。
