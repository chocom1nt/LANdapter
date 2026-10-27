# LANdapter コントリビューションガイド

**言語:** [English](../en/CONTRIBUTING.md) · [Русский](../ru/CONTRIBUTING.md) · [中文](../zh/CONTRIBUTING.md) · [日本語](CONTRIBUTING.md) · [Español](../es/CONTRIBUTING.md)

LANdapter へのバグ報告、修正、ドキュメント更新、新機能の提案を歓迎します。

## Issue を作成する前に

重複を検索し、できる限り現在のコードで再現してください。Go、PostgreSQL、Node.js、OS のバージョン、再現手順、期待値と実際の結果、必要なログや UI スクリーンショットを含めます。ログを添付する前に token、パスワード、個人情報、内部ネットワーク情報を必ず削除してください。

## 開発フロー

```bash
git clone https://github.com/chocom1nt/LANdapter.git
cd LANdapter
git checkout -b fix/short-description
go mod download
cd web && npm ci && cd ..
```

ブランチ名は `fix/...`、`feature/...`、`docs/...` を推奨します。PR は焦点を絞り、変更理由を説明してください。

## テストとフォーマット

```bash
gofmt -w ./cmd ./internal ./storage
go test ./...
make build
cd web && npm run build
```

golangci-lint がある場合は `golangci-lint run ./...` を実行できます。`go.mod` に記載の Go バージョンを使用してください。実行していないテストやビルドを成功したと報告しないでください。

## 規約

- Go: `gofmt`、明示的なエラー処理、秘密情報をログに出さない、公開 API のコメント。
- React: 関数コンポーネント、分かりやすい命名、API エラー処理。
- SQL: `NNN_description.up.sql` と対応する `.down.sql` を追加し、公開済み migration を破壊的に変更する場合は理由を明記。
- API/プロトコル変更時は `README.API.md` とロシア語・中国語・日本語・スペイン語の各版を更新。
- 設定変更時は `CONFIG.md` と各翻訳を更新。
- 秘密情報、ビルド成果物、`node_modules` をコミットしない。

## Pull Request

課題、解決方法、実行したテスト、必要な migration/設定、互換性への影響を説明します。UI 変更には実データを含まないスクリーンショットを添付してください。セキュリティ変更には脅威モデルを記述し、可能であれば回帰テストを追加してください。
