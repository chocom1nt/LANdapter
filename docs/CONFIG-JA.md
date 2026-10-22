# LANdapter 設定ガイド

**言語:** [English](CONFIG.md) · [Русский](CONFIG-RU.md) · [中文](CONFIG-ZH.md) · [日本語](CONFIG-JA.md) · [Español](CONFIG-ES.md)

LANdapter は作業ディレクトリからの相対パス `configs/` にある YAML を読み込みます。対応する環境変数は YAML の値を上書きします。設定は起動時に読み込まれるため、変更後はプロセスを再起動してください。

## Master: `configs/master.yaml`

```yaml
host: "0.0.0.0"
http_port: 8080
ws_port: 8081
db:
  host: "localhost"
  port: 5432
  user: "landapter"
  password: ""
  dbname: "landapter"
  sslmode: "disable"
```

| キー | 説明 | 一般的な値 |
|---|---|---|
| `host` | Master の待ち受けアドレス。`0.0.0.0` は全インターフェース。 | `0.0.0.0` |
| `http_port` | REST API と Web UI の HTTP ポート。 | `8080` |
| `ws_port` | Agent 用 WebSocket ポート。 | `8081` |
| `db.host`, `db.port` | PostgreSQL のホストとポート。 | `localhost`, `5432` |
| `db.user`, `db.password` | PostgreSQL の認証情報。パスワードは `DB_PASSWORD` から渡すことを推奨。 | `landapter`、サンプルは空欄 |
| `db.dbname` | データベース名。 | `landapter` |
| `db.sslmode` | PostgreSQL TLS モード (`disable`、`require`、`verify-ca`、`verify-full`)。 | 開発時のみ `disable` |

Master で使用できる環境変数: `MASTER_HOST`、`HTTP_PORT`、`WS_PORT`、`DB_HOST`、`DB_PORT`、`DB_USER`、`DB_PASSWORD`、`DB_NAME`、`DB_SSLMODE`。管理用 `LANDAPTER_ADMIN_TOKEN` も必須です。DB パスワードとは別の、32 文字以上のランダム値を使用してください。Compose は `.env` から宣言済みの変数をコンテナへ渡します。Go プログラムをホストで直接起動する場合は、起動するプロセスの環境に変数を設定します。

## Agent: `configs/agent.yaml`

```yaml
master_host: "192.168.1.100"
master_port: 8081
master_http_port: 8080
master_scheme: "ws"
uuid: "<agent-id-issued-by-master>"
token: "<one-time-agent-token>"
uuid_file: "./agent.uuid"
installer_args:
  ".exe": "/S"
```

| キー | 説明 |
|---|---|
| `master_host` | Agent から到達可能な Master の IP または DNS 名。 |
| `master_port` | WebSocket ポート。Master の `ws_port`、または TLS プロキシの公開ポートと一致させる。 |
| `master_http_port` | ファイルの HTTP/HTTPS ダウンロードに使うポート。TLS プロキシ経由なら `master_port` と同じ値にできる。 |
| `master_scheme` | TLS のない信頼できる LAN では `ws`、TLS プロキシ経由では `wss`。リモートアクセスには TLS を使用。 |
| `uuid` | Master が発行した Agent ID。設定されている場合、`uuid_file` の値より優先。 |
| `token` | Agent 専用の秘密。必須。DB に平文では保存されない。紛失時は再発行する。 |
| `uuid_file` | `uuid` 未設定時に UUID を保存するファイル。必要に応じて親ディレクトリを作成。 |
| `installer_args` | 拡張子ごとのインストーラ引数。現状 `.exe` はサイレントインストールの引数に使用。 |

Agent token を複数端末で使い回さないでください。同じ UUID に対して token を再発行すると旧 token は無効になります。`configs/agent.yaml` の読み取り権限を制限し、Git やチケット、スクリーンショットに含めないでください。

Agent の環境変数: `MASTER_HOST`、`MASTER_PORT`、`MASTER_HTTP_PORT`、`MASTER_SCHEME`、`AGENT_UUID`、`AGENT_TOKEN`、`UUID_FILE`。`installer_args` などの複雑なマップは YAML に記載してください。

## TLS リバースプロキシ

- ローカル開発: Vite は `http://localhost:3000`、API は `http://localhost:8080`、Agent は `ws://<master>:8081/ws` に接続。
- 本番では HTTPS をリバースプロキシで終端し、`/api/` を内部 `8080`、`/ws` を WebSocket `8081` へ転送します。
- 同じ公開 HTTPS `443` で API と WebSocket を公開する場合、Agent は通常 `master_scheme: "wss"`、`master_port: 443`、`master_http_port: 443` を使用します。
- 内部ポートや DB をインターネットに公開しないでください。信頼できないネットワークを DB 通信が通る場合は、厳格な `db.sslmode` を設定します。

## 設定の検証

Master と Agent は起動時に設定を読み込みます。YAML の構文、整数ポート、作業ディレクトリからの相対パス、環境変数名を確認してください。インストールは [INSTALL-JA.md](INSTALL-JA.md)、本番構成は [DEPLOY-JA.md](DEPLOY-JA.md) を参照してください。
