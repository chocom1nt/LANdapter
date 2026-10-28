# LANdapter インストール・起動ガイド


## 推奨するクイックセットアップ (Docker)

新規のローカル環境では、リポジトリのルートから実行します。

```bash
make setup
make up
make install-frontend
make copy-token
make run-frontend
```

`make setup` は既存の有効なシークレットを変更せずに `.env` の不足項目を補い、ローカル YAML がない場合だけサンプルから作成します。**新しい Docker volume** では PostgreSQL が初期化時に SQL マイグレーションを実行します。既存 volume はこの init スクリプトでは消去も移行もされません。`make copy-token` で管理者トークンをコピーし、Web Settings に一度貼り付けます。`lndp` は `.env` からトークンを自動読込します。

以下の詳細セクションは、手動 PostgreSQL 構成や既存環境向けです。DB volume を削除する意図がない限り `docker compose down -v` は実行しないでください。

**言語:** [English](../en/INSTALL.md) · [Русский](../ru/INSTALL.md) · [中文](../zh/INSTALL.md) · [日本語](INSTALL.md) · [Español](../es/INSTALL.md)

このガイドでは、開発環境またはテスト用ネットワークで LANdapter を起動する手順を説明します。本番環境については [デプロイ](DEPLOY.md) を参照してください。

## 1. 必要なソフトウェア

- バックエンド用 Go **1.26.4 以降**
- PostgreSQL **15 以降**
- React フロントエンド用 Node.js 18 以降と npm
- Git。`make` と PostgreSQL クライアント `psql` も推奨
- Windows では Makefile コマンドに Git Bash または WSL2 を推奨。PowerShell の環境変数構文は異なります。

## 2. ソースコードの取得

```bash
git clone https://github.com/chocom1nt/LANdapter.git
cd LANdapter
go mod download
cp configs/master.yaml.example configs/master.yaml
cp configs/agent.yaml.example configs/agent.yaml
```

`cp` と `export` は Bash の例です。PowerShell では `Copy-Item` と `$env:NAME = "value"` を使用してください。

## 3. データベースの作成

PostgreSQL に LANdapter 専用ユーザーとデータベースを作成します。

```sql
CREATE USER landapter WITH PASSWORD 'replace-with-a-strong-db-password';
CREATE DATABASE landapter OWNER landapter;
```

PostgreSQL が起動しており、Master の設定で指定したホストとポートから接続できることを確認してください。DB ユーザーには LANdapter のスキーマを作成・変更する権限が必要です。

## 4. 秘密情報の設定とマイグレーション

Master は `configs/master.yaml` を読み込みます。環境変数 `DB_PASSWORD` は YAML の DB パスワードを上書きします。`LANDAPTER_ADMIN_TOKEN` は別個の管理者用シークレットで、32 文字以上のランダム値が必要です。DB パスワードと同じ値を使わないでください。

```bash
export DB_PASSWORD='your-database-password'
export LANDAPTER_ADMIN_TOKEN="$(openssl rand -hex 32)"
make migrate-up
```

シークレットはシークレット管理ツール、またはアクセス権を制限した環境ファイルに保存し、Git に登録しないでください。`make migrate-up` は `005_agent_credentials` を含む番号順の SQL マイグレーションを適用します。既存環境をアップグレードする前に PostgreSQL をバックアップしてください。`make migrate-down` はカラムやデータを削除する場合があるため、通常の復旧操作として実行しないでください。

## 5. Master のビルドと起動

```bash
go mod download
make build
./bin/master
```

または `go run cmd/master/main.go` を実行します。Master は既定で HTTP API を 8080、WebSocket を 8081 ポートで待ち受けます。Master プロセスに `DB_PASSWORD` と `LANDAPTER_ADMIN_TOKEN` を渡してください。起動時に DB マイグレーションは自動適用されないため、先に DB を準備する必要があります。

## 6. Web フロントエンド

別ターミナルで実行します。

```bash
cd web
npm ci
npm run dev
```

`http://localhost:3000` を開きます。Vite 開発サーバーは `/api` リクエストを `http://localhost:8080` に転送します。開発サーバーを本番公開に使わないでください。本番用には `npm run build` を実行し、`web/dist/` を Nginx、Caddy などで配信して `/api/` を Master にプロキシします。

Settings 画面に、Master プロセスで使用しているものと同じ `LANDAPTER_ADMIN_TOKEN` を入力して接続設定を保存します。

## 7. Agent の登録と起動

1. Web UI の Settings で Agent ごとの認証情報を発行します。
2. 発行レスポンスをすぐに保存します。Agent の秘密 token が表示されるのは **一度だけ** です。
3. 対象端末の `configs/agent.yaml` で `agent_id` を `uuid` に、token を `token` に設定します。`master_host`、`master_port`、`master_http_port` も合わせます。信頼できる LAN では `master_scheme: "ws"`、TLS プロキシ経由では `"wss"` を使用します。
4. Agent サービスの実行アカウントだけが設定ファイルを読めるようにします。
5. Agent を起動します。

```bash
./bin/agent
# または
go run cmd/agent/main.go
```

Agent は WebSocket ポートと HTTP ファイルダウンロードポートの両方にアクセスする必要があります。Windows のドライバインストールには管理者権限が必要な場合があり、Linux のパッケージインストールにも昇格権限が必要な場合があります。

## 8. Docker Compose

`docker-compose.yml` が起動するのは PostgreSQL と Master のみです。**Web UI と Agent は起動しません。**

```bash
cp .env.example .env
cp configs/master.yaml.example configs/master.yaml
# .env に互いに異なるランダムな秘密を設定
# ホストの 5432 ポートが空いていることを確認
docker compose up -d postgres
```

Master を起動する前に、コンテナの DB にマイグレーションを適用します。Compose の PostgreSQL ポートは既定でホストの loopback に公開されるため、`.env` を読み込み、Makefile が使う `DB_PASSWORD` に `POSTGRES_PASSWORD` を設定します。

```bash
set -a
. ./.env
set +a
export DB_PASSWORD="$POSTGRES_PASSWORD"
make migrate-up
```

または `psql` で `migrations/*.up.sql` を番号順に適用してください。その後:

```bash
docker compose up -d master
docker compose logs -f master
```

Compose のポートは既定で `127.0.0.1` にバインドされます。信頼できないネットワークに直接公開しないでください。Web UI は別途起動し、Agent は各管理端末に配置します。

## 9. 動作確認

```bash
curl -H "Authorization: Bearer $LANDAPTER_ADMIN_TOKEN" \
  http://localhost:8080/api/v1/clients
```

空の DB では通常 `[]` が返ります。Agent のログに接続完了とコマンド待機のメッセージが出ることを確認してください。API 詳細は [API](README.API.md) を参照してください。

## トラブルシューティング

- **DB 接続エラー:** PostgreSQL の稼働状態、ホスト、ポート、ユーザー、パスワード、DB 名とマイグレーションを確認します。
- **Master が起動しない:** `configs/master.yaml` が存在するか、Master の環境に `DB_PASSWORD` があるか、`LANDAPTER_ADMIN_TOKEN` が 32 文字以上か確認します。
- **Agent が接続しない:** `master_host`、各ポート、ファイアウォール、UUID と token の組み合わせを確認します。
- **ファイルをダウンロードできない:** WebSocket に加えて `master_http_port` に接続できる必要があります。
- **UI が 401 を返す:** Settings に保存した管理者 token と Master の環境変数を一致させます。
- **別 OS でコピーした `node_modules` によりビルド失敗:** 対象 OS で `npm ci` を実行してください。

次に [設定](CONFIG.md)、[デプロイ](DEPLOY.md)、[API](README.API.md) を参照してください。
