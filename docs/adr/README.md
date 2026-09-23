# Architecture Decision Records

このプロジェクトの設計判断の記録です。書き方は [0001](0001-record-architecture-decisions.md) を参照してください。新しい ADR は [テンプレート](0000-template.md) をコピーして作成します。

| # | タイトル | ステータス |
|---|---------|-----------|
| [0001](0001-record-architecture-decisions.md) | 設計判断を ADR として記録する | 承認済み |
| [0002](0002-build-custom-app.md) | 既存ツールを組み合わせず専用アプリとして実装する | 承認済み |
| [0003](0003-use-go.md) | 実装言語に Go を採用する | 承認済み |
| [0004](0004-use-wails-react.md) | GUI に Wails v2 + React / TypeScript を採用する | 承認済み |
| [0005](0005-startup-and-tray.md) | Windows サービスではなくログイン時起動のトレイ常駐アプリにする | 承認済み |
| [0006](0006-one-way-sync.md) | 同期は Windows → Immich の一方向とし、ローカルの削除は反映しない | 承認済み |
| [0007](0007-use-sqlite-modernc.md) | 同期状態の保存に SQLite（modernc.org/sqlite）を使う | 承認済み |
| [0008](0008-assets-table-schema.md) | 同期状態を assets テーブルで「現在の状態」として管理する | 承認済み（一部 0009 で変更） |
| [0009](0009-no-local-checksum.md) | ローカルでチェックサムを持たず、同期済みかどうかをパスで判定する | 承認済み |
| [0010](0010-serialize-db-writes.md) | SQLite への書き込みはコネクション 1 本で直列化し、MarkAsSyncing は UPSERT にする | 承認済み |
| [0011](0011-immich-client-design.md) | Immich API クライアントの設計 | 承認済み |
| [0012](0012-app-as-mediator.md) | Syncer と Watcher を独立させ、App を仲介役にする | 承認済み |
| [0013](0013-auto-exclude-new-subfolders.md) | 監視中に作成された新しいサブフォルダは自動で除外リストに入れる | 承認済み |
| [0014](0014-startup-registry.md) | スタートアップ登録はレジストリの Run キーで行う | 承認済み |
| [0015](0015-dev-prod-build-tags.md) | dev ビルドと本番ビルドの切り替えにビルドタグを使う | 承認済み |
| [0016](0016-data-in-appdata.md) | 設定・DB・ログを %APPDATA% 配下に保存する | 承認済み |
| [0017](0017-testing-with-fakes.md) | 外部環境に触れる処理は差し替え可能にし、テストではフェイクを注入する | 承認済み |

## 今後の検討事項（未決定）

- アップロード待ち（`pending`）状態を経由するフロー。誤って監視対象にした写真がアップロードされるのを防ぎ、大量にアップロードする前に除外しやすくする
- ローカルでの削除を Immich に反映するかどうかのオプション化（[0006](0006-one-way-sync.md)）
- EXIF から撮影日時を取得する（[0011](0011-immich-client-design.md)）
- 失敗したアップロードのリトライ強化
- DB ファイルを `%LOCALAPPDATA%` へ移す（[0016](0016-data-in-appdata.md)）
