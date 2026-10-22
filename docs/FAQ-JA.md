# LANdapter よくある質問

**言語:** [English](FAQ.md) · [Русский](FAQ-RU.md) · [中文](FAQ-ZH.md) · [日本語](FAQ-JA.md) · [Español](FAQ-ES.md)

## 基本情報

### LANdapter とは？

Master から Agent を管理し、Windows/Linux 端末にドライバやアプリを展開するクライアント–サーバー型ツールです。

### 対応 OS は？

Master は Windows と Linux でビルドできます。現行 Agent インストーラは Windows の `.exe`、`.msi`、`.inf`、Linux の `.deb`、`.run`、対応する tar アーカイブを扱います。macOS でのインストール実行は未実装です。

### 認証はどうなっていますか？

管理用 REST API には `Authorization: Bearer <LANDAPTER_ADMIN_TOKEN>` が必要です。Master プロセスには 32 文字以上のランダムな `LANDAPTER_ADMIN_TOKEN` を設定します。各 Agent は Settings で発行する個別資格情報を使います。信頼できないネットワークに API を直接公開せず、TLS/WSS とネットワーク制限を利用してください。

### Master は Web UI を配信しますか？

いいえ。開発時は Vite が通常 `http://localhost:3000` で UI を配信します。本番では `web/dist/` をビルドし、Web サーバーまたはリバースプロキシから配信してください。

## インストールと設定

### ポートを変更するには？

`configs/master.yaml` の `http_port` と `ws_port` を変更します。Agent の `master_port` と必要に応じて `master_http_port` も更新し、必要なポートだけをファイアウォールで許可します。

### DB パスワードを YAML に書かずに設定できますか？

Master の環境変数 `DB_PASSWORD` を使ってください。管理用には別の `LANDAPTER_ADMIN_TOKEN` を使用します。

### Agent が接続できない場合は？

Master の待ち受け、`master_host`、`master_port`、`master_http_port`、`master_scheme`、ファイアウォール、UUID と token の組み合わせを確認します。`wss` 使用時は TLS proxy が WebSocket と HTTPS ダウンロードを正しく転送する必要があります。

### WebSocket は接続できるのにファイルをダウンロードできないのはなぜ？

Agent は別途 `master_http_port` を使います。ダウンロード要求では Agent token と `X-Agent-ID` の UUID も検証されます。

### PostgreSQL なしで Master を実行できますか？

現行実装ではできません。PostgreSQL に Agent、ジョブ、結果を保存します。

## 利用方法と制限

### 対応インストール形式は？

Windows: `.exe`、`.msi`、`.inf`。Linux: `.deb`、`.run`、`.tar`、`.tar.gz`、`.tar.bz2`、`.tgz`。配布前にパッケージと引数を確認してください。

### quiet と interactive の違いは？

`quiet` はインストーラの既定のサイレント引数を使用します。`interactive` はインストーラが対応する基本的な UI モードを要求します。細部はパッケージにより異なります。

### Agent 再接続時に pending ジョブは自動実行されますか？

現在は自動実行されません。結果が `pending` として保存される場合がありますが、再接続時の自動再配信は未実装です。

### Wake-on-LAN はどのように動きますか？

Master が端末の MAC 宛てにマジックパケットを送信します。NIC、BIOS/UEFI、OS、ネットワークで WOL が有効である必要があり、ルーティングされた別サブネットでは動作しない場合があります。

## トラブルシューティング

- **認証未設定のエラー:** Master の環境変数 `LANDAPTER_ADMIN_TOKEN` を設定して再起動します。
- **HTTP 401:** UI/API の管理 token と Master の環境変数が一致していません。
- **Agent 認証時の HTTP 503:** Master から PostgreSQL へ接続できるか確認します。
- **Agent が offline:** Agent ログ、ネットワーク、token の再発行有無を確認します。
- **ファイル一覧が空/エラー:** サービスが `uploads/` を読み書きできるか確認します。
- **`.inf` インストール失敗:** Windows で Agent を管理者として実行し、ドライバ要件とコマンド出力を確認します。
- **フロントエンドのネットワークエラー:** `/api/` proxy、Master のホスト/ポート、ブラウザーコンソールを確認します。

詳細: [INSTALL-JA.md](INSTALL-JA.md)、[CONFIG-JA.md](CONFIG-JA.md)、[README.API-JA.md](README.API-JA.md)。
