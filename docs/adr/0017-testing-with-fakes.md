# 0017. 外部環境に触れる処理は差し替え可能にし、テストではフェイクを注入する

- ステータス: 承認済み
- 日付: 2026-07-25

## 背景

- このアプリは次の外部環境に依存している
  - レジストリ
  - 名前付き Mutex
  - `%APPDATA%`
  - Immich サーバー
- テストや動作確認のたびに、開発者の環境を書き換えてしまうのは避けたい
  - 実際のスタートアップ登録
  - 本番のデータ
  - Immich のライブラリ

## 決定

- 外部環境に触れる処理は、インターフェースや関数型のフィールドとして切り出す
- テストでは、そこにフェイクを注入する
  - `internal/startup`: `registryKey` インターフェースと `Startup.openKey`
    - `registry.Key` はメソッドがすべて値レシーバーなので、コードを変えずにこのインターフェースを満たす
  - `internal/singleinstance`: `createMutexFunc`
- `%APPDATA%` に依存する処理は、`t.Setenv("APPDATA", t.TempDir())` で一時ディレクトリに差し替える
- Immich API は `httptest` で差し替える
- 手動で動作を確認するときは、スタブサーバー `tools/immich-stub` を使う
  - `/api/assets` だけを模倣する
  - ディスクに何も書き込まないため、後片付けが要らない
- アサーションには `github.com/stretchr/testify`（`require` / `assert`）を使う

## 結果

- テストを実行しても、開発者の環境を汚さない
- 本番コードに差し替え用の間接層が少し増えるが、受け入れる
