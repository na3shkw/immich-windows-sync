# 0010. SQLite への書き込みはコネクション 1 本で直列化し、MarkAsSyncing は UPSERT にする

- ステータス: 承認済み
- 日付: 2026-07-04

## 背景

- `SyncAssets` はワーカープール（複数の goroutine）でアップロードする
- 各ワーカーは次の順に DB へ書き込む
  1. `MarkAsSyncing`
  2. アップロード
  3. `MarkAsSuccess` または `MarkAsFailed`
- この構成で、レコードが作成されない不具合が 2 つ起きた
  - **UPDATE の空振り**
    - 同期を開始した時点では、レコードが存在しないことがあった
    - `UPDATE` だけで書いた `MarkAsSyncing` が空振りした
    - その結果、後続の `MarkAsSuccess` / `MarkAsFailed` も空振りした
  - **`SQLITE_BUSY`**
    - 複数のコネクションから同時に書き込むと `SQLITE_BUSY` が発生した
    - `MarkAsSyncing` のエラーが無視されていたため、表面化しなかった
    - `internal/syncer` のユニットテストで発覚した

## 決定

- `db.MarkAsSyncing` は `INSERT ... ON CONFLICT(path) DO UPDATE` で実装する
- `db.NewClient` で `sql.DB.SetMaxOpenConns(1)` を設定する
  - Go 側のコネクションプールで書き込みを直列化する

## 検討した選択肢

- **`SQLITE_BUSY` のときにリトライする**
  - エラー処理が散らばる
  - リトライ回数や待ち時間の調整も必要になる

## 結果

- 書き込みの競合がなくなる
- 書き込みは直列になるが、問題にはならない
  - ボトルネックはアップロード（ネットワーク）のため
