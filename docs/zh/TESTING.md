# 测试与验证

请在干净的 checkout 中使用与 CI 相同的 Go/Node 版本。仅仅能打开网页并不代表版本已通过验证。

## 前置条件

- `go.mod` 要求的 Go 版本（当前为 1.26.4 或更高）。
- Node.js 18+、npm，以及从 lockfile 安装的依赖。
- 用于手动集成验证的 Docker Compose 和一次性 PostgreSQL 数据库。

## 自动检查

```bash
go test ./...
make test
cd web && npm ci && npm test && npm run build
make test-cover
```

## 手动冒烟测试

1. 启动 PostgreSQL 和 Master，检查迁移、日志和健康状态。
2. 打开 Web UI，使用管理员 token 认证，为 Agent 发放独立凭据并连接真实测试 Agent。
3. 检查 online/offline、设备清单、CPU/RAM/uptime、文件上传/下载及安全的测试安装程序。
4. 在一次性测试客户端上验证策略下发和计划任务重复运行。
5. 测试 `lndp` 标签页、搜索、选择、窄终端布局、上传以及安装/删除确认。
6. 在数据库副本上测试备份/恢复和升级迁移，切勿只在唯一生产数据库上测试。

## 已知缺口

- 当前尚无完整的数据库集成测试套件。
- Windows Agent 应在 Windows 上验证，Linux Agent 应在 Linux 上验证；交叉编译不能验证平台行为。
- 数据库中的演示记录不能证明真实 Agent 可以完成认证或返回设备清单。
