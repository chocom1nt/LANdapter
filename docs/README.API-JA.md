# API ドキュメント

**言語:** [English](README.API.md) · [Русский](README.API-RU.md) · [中文](README.API-ZH.md) · [日本語](README.API-JA.md) · [Español](README.API-ES.md)

## 概要

LANdapter は、クライアント管理とソフトウェア配布のための RESTful HTTP API と、エージェントとのリアルタイム通信用 WebSocket エンドポイントを提供します。HTTP API は `master.yaml` で設定したポート（既定 `8080`）で提供されます。WebSocket は別ポート（既定 `8081`）です。

> **認証**: 現行バージョンでは認証を実装していません。信頼できる内部ネットワーク向けです。追加のセキュリティ対策なしに API をインターネットに公開しないでください。

---

## ベース URL

- HTTP API: `http://<master-host>:<http-port>/api/v1/`
- WebSocket: `ws://<master-host>:<ws-port>/ws`

---

## HTTP API エンドポイント

すべてのエンドポイントは JSON を返します。エラーは適切な HTTP ステータスコードとプレーンテキストのメッセージで返されます。

---

### `GET /api/v1/clients`

登録済みクライアントの一覧を取得します。

**クエリパラメータ**

| 名前     | 型      | 説明 |
|----------|---------|------|
| `online` | boolean | 任意。オンライン状態でフィルタ（`true` または `false`）。省略時は全クライアント。 |

**レスポンス** – `200 OK`

```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "hostname": "pc-01",
    "os": "windows",
    "mac": "00:11:22:33:44:55",
    "online": true,
    "last_seen": "2025-01-15T10:30:00Z"
  }
]
```

**エラー**

- `500 Internal Server Error` – データベースクエリ失敗。

---

### `POST /api/v1/install`

新しいインストールジョブを作成し、選択したオンラインエージェントにコマンドを送信します。

**リクエストボディ** – `application/json`

```json
{
  "file_ids": ["uuid1", "uuid2"],
  "client_ids": ["client-uuid1", "client-uuid2"],
  "mode": "quiet"
}
```

| フィールド   | 型       | 説明 |
|--------------|----------|------|
| `file_ids`   | []string | ファイル ID のリスト（`/api/v1/upload` で事前アップロード）。 |
| `client_ids` | []string | インストール先クライアント UUID のリスト。 |
| `mode`       | string   | インストールモード: `"quiet"`（サイレント）または `"interactive"`（UI あり）。既定は `"quiet"`。 |

**レスポンス** – `200 OK`

```json
{
  "job_id": "job-uuid"
}
```

ジョブが作成されキューに入ります。オンラインのクライアントには即座にコマンドが送られ、オフラインの場合は次回接続時に受信します。

**エラー**

- `400 Bad Request` – 必須フィールド欠落または無効な JSON。
- `500 Internal Server Error` – データベースまたはストレージエラー。

---

### `POST /api/v1/upload`

マスターサーバーにファイルをアップロードします。ファイルは一意名で `uploads/` に保存され、メタデータ（元の名前、サイズ、アップロード時刻）も保存されます。

**リクエスト** – フィールド名 `file` の `multipart/form-data`。

**レスポンス** – `200 OK`

```json
{
  "file_id": "generated-uuid",
  "name": "original_filename.exe"
}
```

**エラー**

- `400 Bad Request` – ファイルフィールド欠落またはサイズ超過（上限 100 MB）。
- `500 Internal Server Error` – ディスク書き込みエラー。

---

### `GET /api/v1/files`

アップロード済みファイルとメタデータの一覧を取得します。

**レスポンス** – `200 OK`

```json
[
  {
    "id": "file-uuid",
    "name": "driver.exe",
    "type": ".exe",
    "size": 10485760,
    "uploadedAt": "2025-01-15T12:00:00Z",
    "version": "1.0.0",
    "description": "Network driver"
  }
]
```

**エラー**

- `500 Internal Server Error` – アップロードディレクトリを読めない。

---

### `DELETE /api/v1/files/{id}`

サーバーからファイル（バイナリとメタデータ）を削除します。

**レスポンス** – `200 OK`

```json
{
  "status": "deleted"
}
```

**エラー**

- `400 Bad Request` – 無効なファイル ID。
- `404 Not Found` – ファイルが存在しない。
- `500 Internal Server Error` – 削除失敗。

---

### `GET /api/v1/clients/{id}/devices`

特定クライアントに接続されたハードウェアデバイスの一覧を要求します。マスターは WebSocket で転送し、最大 5 秒エージェントの応答を待ちます。

**レスポンス** – `200 OK`

形式はエージェントの OS により異なります:
- **Windows**: `Get-PnpDevice | ConvertTo-Json` の JSON 配列。
- **Linux**: `lsusb`、`lspci`、`lscpu` のプレーンテキスト出力。

例（Windows）:

```json
[
  {
    "FriendlyName": "Intel(R) USB 3.1",
    "Class": "USB",
    "Status": "OK"
  }
]
```

**エラー**

- `400 Bad Request` – 無効なクライアント UUID。
- `408 Request Timeout` – 5 秒以内に応答なし。
- `500 Internal Server Error` – WebSocket コマンド失敗。

---

### `GET /api/v1/clients/{id}/stats`

クライアントのシステム統計（CPU、メモリ、稼働時間）を取得します。レスポンス構造はプラットフォーム依存です。

**レスポンス** – `200 OK`

**Windows の例**:

```json
{
  "cpu_percent": 12.5,
  "mem_available_mb": 4096,
  "mem_total_mb": 8192,
  "uptime_seconds": 3600,
  "uptime_human": "1:00:00"
}
```

**Linux の例**:

```json
{
  "cpu_percent": 8.2,
  "mem_used_mb": 2048,
  "mem_total_mb": 4096,
  "uptime_seconds": 7200
}
```

**エラー** – `/devices` と同様。

---

### `POST /api/v1/wol`

選択したクライアント（MAC アドレス）に Wake-on-LAN マジックパケットを送信します。マスターはクライアント ID から MAC を引き、ポート 9 に UDP ブロードキャストします。

**リクエストボディ**

```json
{
  "client_ids": ["uuid1", "uuid2"]
}
```

**レスポンス** – `200 OK`

```json
{
  "status": "sent"
}
```

**エラー** – 特定の HTTP エラーなし。エラーはログに記録されますが HTTP レスポンスには影響しません。

---

### `POST /api/v1/parse-driver`（スタブ）

Web サイトからドライバを検出するためのプレースホルダー。現在は静的リストを返します。

**リクエストボディ**

```json
{
  "url": "https://example.com/drivers"
}
```

**レスポンス** – `200 OK`

```json
[
  {"name": "Driver1.inf", "url": "http://example.com/driver1.inf"},
  {"name": "Driver2.exe", "url": "http://example.com/driver2.exe"}
]
```

---

## WebSocket プロトコル

マスターは `/ws` で WebSocket エンドポイントを提供し、エージェントとの永続接続を維持します。

### ハンドシェイク

接続後、エージェントは JSON 形式のハンドシェイクを送信する必要があります:

```json
{
  "type": "handshake",
  "uuid": "agent-uuid",
  "hostname": "my-pc",
  "os": "windows",
  "mac": "00:11:22:33:44:55"
}
```

マスターは DB のクライアントレコードを更新または挿入し、オンラインにマークして読み書きループを開始します。

### 受信コマンド（マスター → エージェント）

マスターは次の構造のコマンドを送信します:

```json
{
  "type": "install",
  "job_id": "job-uuid",
  "payload": { ... }
}
```

- `type`: `"install"`、`"stats"`、`"devices"`。
- `install` の payload: `{"files": ["file-id1", ...], "mode": "quiet"}`。
- `stats` と `devices` の payload は空。

### 送信レスポンス（エージェント → マスター）

エージェントは結果メッセージで応答します:

```json
{
  "type": "result",
  "job_id": "job-uuid",
  "status": "success",
  "output": "...",
  "error": "",
  "snapshot_before": { ... },
  "snapshot_after": { ... }
}
```

`stats` と `devices` 要求ではエージェントは次を送信します:

```json
{
  "type": "stats",
  "job_id": "cmd-uuid",
  "data": { ... }
}
```

`data` フィールドに要求された統計またはデバイス一覧が含まれます。

### 接続管理

- マスターは定期的に ping を送信（既定間隔約 54 秒）して接続を維持します。
- エージェントは pong を自動返信（WebSocket ライブラリが処理）。
- 読み取りタイムアウト（120 秒）でマスターは接続を閉じます。
- エージェントは指数バックオフで再接続します。

---

## データモデル

### Client（クライアント）

| フィールド  | 型        | 説明 |
|-------------|-----------|------|
| `id`        | uuid.UUID | 一意 ID |
| `hostname`  | string    | ホスト名 |
| `os`        | string    | OS（windows、linux など） |
| `mac`       | string    | MAC アドレス（WOL 用） |
| `online`    | boolean   | オンライン状態 |
| `last_seen` | timestamp | 最終活動時刻（RFC3339） |

### Job（ジョブ）

| フィールド   | 型        | 説明 |
|--------------|-----------|------|
| `id`         | uuid.UUID | ジョブ ID |
| `files`      | []string  | インストールするファイル ID |
| `created_at` | timestamp | 作成日時 |

### Job Result（ジョブ結果）

| フィールド        | 型        | 説明 |
|-------------------|-----------|------|
| `id`              | uuid.UUID | 結果レコード ID |
| `job_id`          | uuid.UUID | 関連ジョブ |
| `client_id`       | uuid.UUID | 対象クライアント |
| `status`          | string    | `pending`、`running`、`success`、`failed` |
| `output`          | string    | インストール出力（stdout/stderr） |
| `error`           | string    | 失敗時のエラーメッセージ |
| `started_at`      | timestamp | コマンド送信時刻 |
| `finished_at`     | timestamp | 結果受信時刻 |
| `snapshot_before` | JSON      | インストール前のシステム状態 |
| `snapshot_after`  | JSON      | インストール後のシステム状態 |

---

## バージョン

現在の API バージョン: **v1**。すべてのエンドポイントは `/api/v1/` プレフィックス付きです。
