# LANdapter

<p align="center">
  <img src="docs/assets/logo.png" width="128" alt="LANdapter logo">
</p>

<p align="center">
  <img src="https://img.shields.io/badge/go-1.26.4+-00ADD8?style=flat&logo=go&logoColor=white" alt="Go">
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
- **クロスプラットフォーム** – Master と Agent は Windows / Linux に対応します。macOS でのインストール実行は現在未対応です。

---

## クイックスタート

### 要件

- Go 1.26.4+
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
npm ci
cd ..
```

マイグレーション前に、PostgreSQL 管理者として専用ユーザーとデータベースを作成します（例: `psql` 内で実行）。

```sql
CREATE USER landapter WITH PASSWORD 'your-database-password';
CREATE DATABASE landapter OWNER landapter;
```

下の `DB_PASSWORD` には同じパスワードを設定してください。既にユーザーや DB が存在する場合は再作成せず、所有者と権限を確認してください。

マイグレーションや Master の起動前に、PostgreSQL のパスワードと 32 文字以上の管理者用シークレットを設定します:

```bash
export DB_PASSWORD='your-database-password'
export LANDAPTER_ADMIN_TOKEN="$(openssl rand -hex 32)"
```

DB マイグレーション:

```bash
make migrate-up
```

マスターを起動:

```bash
go run cmd/master/main.go
```

別のターミナルでフロントエンドを起動:

```bash
cd web
npm run dev
```

`http://localhost:3000` を開き、Settings で `LANDAPTER_ADMIN_TOKEN` を入力して接続を保存し、エージェント用の個別トークンを発行します。トークンは一度だけ表示されます。`agent_id` と `token` を `configs/agent.yaml` の `uuid` と `token` に保存してください。

認証情報を保存してから、別のターミナルでエージェントを起動します:

```bash
go run cmd/agent/main.go
```

詳細なデプロイ手順は [docs/INSTALL-JA.md](docs/INSTALL-JA.md) を参照してください。

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
docker build -t landapter-master .

# エージェント
docker build -f Dockerfile.agent -t landapter-agent .
```

`docker-compose.yml` が起動するのは **PostgreSQL と Master のみ**です。フロントエンドと Agent は自動起動しません。事前に `configs/master.yaml.example` を `configs/master.yaml` にコピーし、`.env` を設定して SQL マイグレーションを適用してください。詳細は [docs/INSTALL-JA.md](docs/INSTALL-JA.md) を参照してください。フロントエンドは別途起動するか、`web/dist/` を Web サーバーで配信します。

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
├── makefile
├── go.mod
└── README.md
```

---

## ドキュメント

全ドキュメントの索引は [docs/README.md](docs/README.md) です。目的に応じて次のガイドを参照してください。

- [インストール](docs/INSTALL-JA.md)：要件、DB、マイグレーション、Master/UI 起動、Agent 登録。
- [設定](docs/CONFIG-JA.md)：Master/Agent の設定項目と環境変数。
- [アーキテクチャ](docs/ARCHITECTURE-JA.md)：構成要素、資格情報、WebSocket、現状の制限。
- [本番デプロイ](docs/DEPLOY-JA.md)：TLS proxy、サービス保護、バックアップ、更新。
- [API](docs/README.API-JA.md)：REST エンドポイントと WebSocket プロトコル。
- [FAQ](docs/FAQ-JA.md) と [コントリビューション](docs/CONTRIBUTING-JA.md)。

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
