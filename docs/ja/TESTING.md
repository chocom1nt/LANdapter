# テストと検証

クリーンな checkout で CI と同じ Go/Node バージョンを使用してください。Web ページが開くだけではリリース検証になりません。

## 前提条件

- `go.mod` が要求する Go（現在は 1.26.4 以降）。
- Node.js 18 以降、npm、および lockfile からインストールした依存関係。
- 手動統合確認用の Docker Compose と使い捨て PostgreSQL。

## 自動テスト

```bash
go test ./...
make test
cd web && npm ci && npm test && npm run build
make test-cover
```

## 手動スモークテスト

1. PostgreSQL と Master を起動し、マイグレーションとログを確認します。
2. Web UI で管理者トークンを設定し、Agent ごとの認証情報を発行して実 Agent を接続します。
3. online/offline、デバイス一覧、CPU/RAM/uptime、ファイル転送、安全なテストインストーラーを確認します。
4. 使い捨て端末でポリシー配信とスケジュール再実行を確認します。
5. `lndp` のタブ、検索、選択、狭い端末幅、アップロード、インストール/削除確認を試します。
6. 本番 DB ではなくコピーに対してバックアップ/復元と更新マイグレーションを確認します。

## 既知の不足

- DB を使用した統合テスト一式はまだありません。
- Windows Agent は Windows で、Linux Agent は Linux で検証してください。クロスコンパイルだけでは OS 固有の動作を検証できません。
- デモ用 DB レコードは実 Agent の認証やデバイス取得の証明にはなりません。
