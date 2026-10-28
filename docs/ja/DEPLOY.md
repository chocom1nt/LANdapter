# LANdapter 本番環境へのデプロイ

**言語:** [English](../en/DEPLOY.md) · [Русский](../ru/DEPLOY.md) · [中文](../zh/DEPLOY.md) · [日本語](DEPLOY.md) · [Español](../es/DEPLOY.md)

LANdapter はリモートでインストーラを実行します。対象端末や権限を誤るとシステムが変更されるため、最初はテスト端末で確認し、承認済みパッケージのみを配布してください。

## 推奨構成

- 専用 Linux サーバーまたは VM で Master を動かす。
- PostgreSQL は Master と許可された DB 管理者からのみ接続できるようにする。
- Web UI は Nginx/Caddy で配信する。
- TLS リバースプロキシから HTTPS/WSS を公開し、内部 8080/8081 を信頼できないネットワークへ直接公開しない。
- Agent はサービスアカウントで実行し、インストールに必要な最小限の権限だけを与える。

## 秘密情報と DB

互いに異なる 2 つの秘密を生成します。

```bash
openssl rand -hex 32
openssl rand -hex 32
```

一方は PostgreSQL パスワード、もう一方は `LANDAPTER_ADMIN_TOKEN` に使用します。管理用変数がない、または 32 文字未満の場合、Master は起動しません。秘密を Git、ログ、文書、フロントエンドのソースに保存しないでください。管理 token の変更時は Web UI の設定も更新して Master を再起動します。Agent token は発行時に一度だけ表示され、DB には SHA-256 ハッシュのみ保存されます。

## DB とアップグレード

1. アプリ専用の PostgreSQL ユーザーと DB を作成します。
2. ネットワークに応じて `db.sslmode` を設定します。`disable` は制御されたローカル開発向けです。
3. マイグレーション前にバックアップします。
4. 正しい `PG_HOST`、`PG_PORT`、`PG_USER`、`PG_DB`、`DB_PASSWORD` を設定し、`make migrate-up` を実行します。Makefile は `psql` で番号順の SQL を実行します。Master 起動時に自動適用されるわけではありません。
5. アプリとマイグレーションのバージョンを記録します。

本番では `.down.sql` がデータやカラムを削除する可能性があるため、内容を確認せず `make migrate-down` を実行しないでください。

## Master サービス

プロジェクト指定の Go バージョンでビルドし、バイナリ、`configs/master.yaml`、永続化された `uploads/` と権限を制限した環境ファイルを配置します。環境ファイルの例:

```dotenv
DB_PASSWORD=replace-with-database-secret
LANDAPTER_ADMIN_TOKEN=replace-with-independent-random-admin-secret
```

設定と `uploads/` は作業ディレクトリからの相対パスなので、サービスの `WorkingDirectory` を設定してください。ログインできない専用ユーザー、再起動ポリシー、ファイル権限、ディスク容量の監視を設定します。

## TLS リバースプロキシ

`web/dist/` を公開し、SPA の不明なパスは `index.html` に戻します。`/api/` は `http://127.0.0.1:8080` に、正確な `/ws` パスは `http://127.0.0.1:8081` に転送します。WebSocket では HTTP/1.1 と `Upgrade` / `Connection` ヘッダーが必要です。エンドポイントは `/ws` であり、末尾のスラッシュを必須としません。公開 HTTPS ポート 443 に集約する場合、Agent は通常 `master_scheme: "wss"`、`master_port: 443`、`master_http_port: 443` とします。

Nginx の設定例（証明書とドメインは環境に合わせて設定してください）:

```nginx
location /api/ {
    proxy_pass http://127.0.0.1:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Real-IP $remote_addr;
}
location = /ws {
    proxy_pass http://127.0.0.1:8081;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
}
```

リモート Agent から WebSocket とファイルダウンロードの両方をテストしてください。TLS を提供しないプロキシで `wss` を使用しないでください。

## ファイアウォールと公開範囲

- 運用者と Agent 向けには通常 TCP 443 のみ公開します。
- プロキシが同一ホストの場合、内部 8080/8081 は loopback またはプライベートネットワークに限定します。
- PostgreSQL 5432 は許可されたホストだけに公開します。
- 公衆ネットワーク経由の管理には VPN を使用し、DB ポートは公開しないでください。
- 管理 API は `Authorization: Bearer ...` を必要とします。管理 token を信頼できない利用者へ共有しないでください。CORS は認証や TLS の代わりにはなりません。

## Agent とジョブ

Settings で各 Agent の専用認証情報を発行して安全に保存します。Windows のドライバ導入には管理者権限、Linux のパッケージやスクリプトには昇格権限が必要な場合があります。配布前にパッケージの出所、署名/ハッシュ、実行引数を確認し、まずテスト用 Agent に対して実行します。

**現在の制限:** オフライン Agent 向けのジョブは `pending` として保存される場合がありますが、Agent 再接続時の自動再配信は未実装です。スケジュール済みや保証付きの配信に依存しないでください。

## バックアップ・監視・更新

- PostgreSQL を定期的にバックアップし、復元テストを行います。
- `uploads/` をバックアップし、復元時には DB 内のファイル ID と整合させます。
- systemd/コンテナから stdout/stderr を収集し、外部へ送る前に token などを除去します。
- 再起動の繰り返し、DB エラー、ディスク不足、長期間オフラインの Agent を監視します。
- 更新手順: backup → 新バージョン配置 → migration → サービス再起動 → API、UI、テスト Agent を検証。

初回導入は [インストール](INSTALL.md)、設定項目は [設定](CONFIG.md) を参照してください。
