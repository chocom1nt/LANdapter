# LANdapter アーキテクチャ

**言語:** [English](ARCHITECTURE.md) · [Русский](ARCHITECTURE-RU.md) · [中文](ARCHITECTURE-ZH.md) · [日本語](ARCHITECTURE-JA.md) · [Español](ARCHITECTURE-ES.md)

LANdapter は、制御プレーン（Master/API）、管理対象端末（Agent）、Web フロントエンド、PostgreSQL の永続ストレージで構成されます。

```text
ブラウザー ── HTTP/REST + 管理 Bearer token ──> Master :8080 ──> PostgreSQL
Agent ─────── 認証済み WebSocket ─────────────> Master :8081
Agent ─────── Agent token 付き HTTP(S) ダウンロード ─> Master :8080
ブラウザー ── HTTPS ──> TLS proxy ──> 静的 UI + /api と /ws
```

## コンポーネント

- **Master (`cmd/master`)**: YAML と環境変数を読み込み、PostgreSQL に接続し、HTTP と WebSocket サーバーを起動します。
- **HTTP API (`internal/master/handlers.go`)**: クライアント、インストールジョブ、ファイル、デバイス/統計、Wake-on-LAN、Agent 認証情報を管理します。管理 API には `Authorization: Bearer <LANDAPTER_ADMIN_TOKEN>` が必要です。
- **WebSocket (`internal/master/websocket.go`)**: handshake 時に UUID と Agent 固有 token を検証し、コマンドと結果を処理します。
- **Agent (`cmd/agent`、`internal/agent`)**: 接続し、失敗時は間隔を延ばして再試行し、許可されたファイルをダウンロードし、対応 OS でインストーラを実行して結果を返します。
- **Storage (`storage`)**: ストレージインターフェースと PostgreSQL 実装。番号付き SQL migration は `migrations/` にあります。
- **Web (`web/`)**: React SPA。開発時の Vite は通常 3000 ポートで、`/api` をローカル Master に転送します。

## 認証情報の流れ

1. Master プロセスに 32 文字以上のランダム `LANDAPTER_ADMIN_TOKEN` を設定します。
2. 管理 API は `Authorization: Bearer <token>` を使用します。
3. `POST /api/v1/agents/enroll` で Agent UUID と token を新規発行するか、既存 UUID の token を更新します。
4. PostgreSQL には Agent token の SHA-256 ハッシュだけが保存され、元の token は発行応答で一度だけ表示されます。
5. Agent は WebSocket 最初の `handshake` メッセージで UUID と token を提示し、Master はハッシュを定時間比較します。
6. ファイルダウンロードでは `/api/v1/agent/files/{id}` に `Authorization: Bearer <agent-token>` と `X-Agent-ID: <uuid>` を付けます。

認証だけでは通信は暗号化されません。信頼できないネットワークでは TLS/WSS とネットワーク制限を使ってください。

## 通信とジョブ

Agent は `/ws` に接続し、`type`、`uuid`、`token`、`hostname`、`os`、`mac` を含む JSON handshake を送信します。その後、WebSocket でインストール、統計、デバイスなどのコマンドとレスポンスを交換します。正確な形式は [README.API-JA.md](README.API-JA.md) を参照してください。

`POST /api/v1/install` はファイル ID とクライアント ID を検証し、ジョブと端末ごとの結果を作成します。オンライン Agent には即時送信し、結果を PostgreSQL に保存します。オフライン宛てのジョブは `pending` として残る場合がありますが、**再接続時の自動再送はまだ実装されていません**。

対応するインストール形式は Windows `.exe`、`.msi`、`.inf`、Linux `.deb`、`.run`、`.tar`、`.tar.gz`、`.tar.bz2`、`.tgz` です。macOS でのインストール実行は未対応です。

## ファイルとデータ

アップロードファイルは一意な ID で `uploads/` に保存され、JSON メタデータファイルが付随します。このディレクトリを永続化し、アクセスを制限し、PostgreSQL と合わせてバックアップしてください。DB には Agent/クライアント、ジョブ、結果、token ハッシュが保存され、元の Agent token は保存されません。

## 制限とセキュリティ

- Master と Agent は正しい設定、ネットワーク、資格情報を必要とします。
- 管理 API と Agent token 発行は `LANDAPTER_ADMIN_TOKEN` で保護されます。
- Agent ごとに異なる token を利用し、漏えい時は再発行します。古い token は無効になります。
- リモートインストーラ実行は高権限操作です。パッケージ、管理者、ファイル、ネットワーク、OS 権限を制限してください。
- Master CLI、グループポリシー、スケジューラーは現行バージョンでは未実装です。README のロードマップを参照してください。
