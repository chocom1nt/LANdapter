# LANdapter

<p align="center">
  <img src="docs/assets/logo.png" width="128" alt="LANdapter logo">
</p>

<p align="center">
  <img src="https://img.shields.io/badge/go-1.21+-00ADD8?style=flat&logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/react-18.2-61DAFB?style=flat&logo=react&logoColor=white" alt="React">
  <img src="https://img.shields.io/badge/platform-windows%20%7C%20linux-lightgrey" alt="Platform">
  <img src="https://img.shields.io/badge/postgresql-15-4169E1?style=flat&logo=postgresql&logoColor=white" alt="PostgreSQL">
</p>

**言語:** [English](README.md) · [Русский](README-RU.md) · [中文](README-ZH.md) · [日本語](README-JA.md) · [Español](README-ES.md)

**LANdapter** は、ローカルネットワーク上でドライバとソフトウェアを集中リモートインストールするクライアント–サーバーシステムです。

**マスター**（サーバー）と **エージェント**（クライアント端末）で構成されます。マスターは接続されたエージェントを管理し、ファイルを配布し、インストールの進捗を追跡します。エージェントは WebSocket でコマンドを受信し、ファイルをダウンロードして、サイレントまたは対話モードでインストールを実行します。

オフィスや産業ネットワークで、数十～数百台の PC にドライバ更新やアプリ配布を迅速に行うワークステーション管理を簡素化することを目的としています。

---

## 機能

- **集中管理** – すべてのエージェントが単一のマスターに接続。
- **リモートインストール** – EXE、MSI、INF、DEB、RUN、TAR などを配布し、適切な引数で実行。
- **2 つのインストールモード** – サイレント（UI なし）と対話（ユーザーインターフェースあり）。
- **統計収集** – CPU、RAM、稼働時間などのメトリクスとデバイス一覧（PnP、lsusb、lspci）。
- **履歴とレポート** – 各ジョブを DB に保存し、インストール前後のシステムスナップショットを保持。
- **復元ポイント** – Windows でインストール前に自動的に復元ポイントを作成。
- **Web UI** – ダークテーマの React ダッシュボード、ファイルライブラリ、インストールウィザード。
- **柔軟な設定** – YAML と環境変数。
- **クロスプラットフォーム** – マスターとエージェントは Windows / Linux（エージェントは macOS も限定的にサポート）。

---

## クイックスタート

### 要件

- Go 1.21+
- PostgreSQL 15+
- Node.js 18+（フロントエンド）
- Make（任意）

### インストールと起動

リポジトリをクローン:

```bash
git clone https://github.com/chocom1nt/LANdapter.git
cd LANdapter
```

Go 依存関係:

```bash
go mod download
```

フロントエンド依存関係:

```bash
cd web
npm install
cd ..
```

DB マイグレーション:

```bash
make migrate-up
# または手動:
psql -h localhost -U postgres -d landapter -f migrations/001_init.up.sql
psql -h localhost -U postgres -d landapter -f migrations/002_add_mac_up.sql
psql -h localhost -U postgres -d landapter -f migrations/003_add_devices_up.sql
psql -h localhost -U postgres -d landapter -f migrations/004_add_snapshots.up.sql
```

マスターを起動:

```bash
go run cmd/master/main.go
```

エージェントを起動（別ターミナル）:

```bash
go run cmd/agent/main.go
```

フロントエンドを起動（3 つ目のターミナル）:

```bash
cd web
npm run dev
```

ブラウザで `http://localhost:3000` を開きます。

詳細なデプロイ手順は [docs/INSTALL.md](docs/INSTALL.md) を参照してください。

---

## ソースからのビルド

### Go バイナリ

```bash
make build
```

または:

```bash
go build -o bin/master cmd/master/main.go
go build -o bin/agent cmd/agent/main.go
```

### フロントエンド

```bash
cd web
npm run build
```

静的ファイルは `web/dist/` に出力されます。

### Docker イメージ

```bash
# マスター
docker build -f Dockerfile.master -t landapter-master .

# エージェント
docker build -f Dockerfile.agent -t landapter-agent .
```

または PostgreSQL 付きのフルスタック:

```bash
docker-compose up -d
```

---

## プロジェクト構成

```text
LANdapter/
├── cmd/                  # エントリポイント
│   ├── master/
│   └── agent/
├── internal/             # 内部パッケージ
│   ├── common/           # 共通型、ログ、設定
│   ├── master/           # マスター（HTTP、WebSocket、ハンドラ）
│   └── agent/            # エージェント（接続、インストール）
├── storage/              # データ層（インターフェース + PostgreSQL）
├── migrations/           # SQL マイグレーション
├── web/                  # React フロントエンド
├── docs/                 # ドキュメント
├── configs/              # 設定サンプル
├── uploads/              # アップロード（自動作成）
├── docker-compose.yml
├── Makefile
├── go.mod
└── README.md
```

---

## ドキュメント

[docs/](docs/) に詳細があります:

- [API リファレンス](docs/README.API-JA.md) – REST / WebSocket（[English](docs/README.API.md) · [Русский](docs/README.API-RU.md) · [中文](docs/README.API-ZH.md) · [Español](docs/README.API-ES.md)）。
- [インストールガイド](docs/INSTALL.md) – デプロイ手順。
- [設定](docs/CONFIG.md) – `master.yaml` / `agent.yaml`。
- [アーキテクチャ](docs/ARCHITECTURE.md) – 設計、ジョブライフサイクル、WebSocket。
- [本番デプロイ](docs/DEPLOY.md) – サービス、プロキシ、監視。
- [FAQ](docs/FAQ.md) – よくある質問。

---

## テスト

ユニットテスト:

```bash
make test
```

統合テスト（実 DB が必要）:

```bash
make test-integration
```

カバレッジ:

```bash
make test-cover
```

---

## ロードマップ

- グループポリシー（グループでクライアント選択）
- マスター CLI（Web UI なし運用）
- Active Directory / LDAP 連携
- 追加パッケージ形式（AppImage、Flatpak）
- スケジュールジョブ
- ベンダーサイトからのドライバ解析の改善

---

## ライセンス

MIT ライセンス。詳細は [LICENSE](LICENSE) を参照してください。

---

質問や提案は [Issue](https://github.com/chocom1nt/LANdapter/issues) または Pull Request でお知らせください。
