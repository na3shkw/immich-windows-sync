# AGENTS.md

このリポジトリで作業する AI コーディングエージェント向けのガイドです。プロジェクトの概要・技術スタック・ディレクトリ構成・セットアップ手順は [README.md](README.md) を、設計判断の経緯は [docs/adr/](docs/adr/README.md) を参照してください。ここには作業時に守るべきルールをまとめています。

## コマンド

| 目的 | コマンド |
|------|---------|
| 開発モード起動 | `wails dev` |
| 本番ビルド | `wails build` |
| Go テスト | `go test ./internal/...` |
| 特定パッケージのテスト | `go test ./internal/config/...` |
| フロントエンドビルド | `cd frontend && npm run build` |
| Immich スタブ起動 | `go run ./tools/immich-stub -addr 127.0.0.1:2283` |

- `go test` はパッケージ（ディレクトリ）単位で実行する。個別の `.go` ファイルを指定すると、同じパッケージ内の他のファイルが見えず `undefined` エラーになる
- 動作確認で実際の Immich にアップロードしたくない場合は `tools/immich-stub` を使う

## 守るべき制約

詳細と理由はリンク先の ADR を参照。

- Tailwind CSS は v3 から上げない。`frontend/wailsjs/` は自動生成物なので手で編集しない（[0004](docs/adr/0004-use-wails-react.md)）
- SQLite ドライバは `modernc.org/sqlite` を使い、CGO に依存しない（[0007](docs/adr/0007-use-sqlite-modernc.md)）
- `db.NewClient` の `SetMaxOpenConns(1)` と、`MarkAsSyncing` の UPSERT を崩さない（[0010](docs/adr/0010-serialize-db-writes.md)）
- タイムスタンプは UTC で保存する。`updated_at` は SQL 内で `DATETIME('now')` を指定する（[0008](docs/adr/0008-assets-table-schema.md)）
- `Syncer` と `Watcher` は互いを参照しない。つなぎ込みは `app.go` で行う。Watcher のチャネルを読む goroutine を `StartWatcher` の中で起動しない（[0012](docs/adr/0012-app-as-mediator.md)）
- `SaveConfig` には `a.cfg` を書き換えたものではなく、新しく組み立てた `config.Config` を渡す（[0013](docs/adr/0013-auto-exclude-new-subfolders.md)）
- データフォルダのパスは必ず `appenv.AppDir()` を通して取得する（[0015](docs/adr/0015-dev-prod-build-tags.md)）
- テストで実際のレジストリ・ユーザーデータ・Immich を書き換えない。外部環境はフェイクに差し替える（[0017](docs/adr/0017-testing-with-fakes.md)）

## テストの書き方

- アサーションは `github.com/stretchr/testify` を使う。失敗したらテストを打ち切りたい検証は `require`、続けたい検証は `assert`
- `%APPDATA%` に依存する処理は `t.Setenv("APPDATA", t.TempDir())` で一時ディレクトリに差し替える
- ラウンドトリップ系のテスト（Save → Load など）は、`assert.Equal(t, want, got)` で構造体をまるごと比較する

## 規約

- コミットメッセージは**日本語**で書く
- コードコメントは日本語で書く。既存コードのコメントの量・命名に合わせる
- 抽象化は同じパターンが 3 回現れてから行う（Rule of 3）。それまでは明示的な重複を許容する
- 設計上の判断をしたら、`docs/adr/` に ADR を追加する（形式は [`0000-template.md`](docs/adr/0000-template.md)）

## ドキュメントの書き方

README・AGENTS.md・ADR などのドキュメントは、次のように書く。

- 箇条書きは一文一項目にする
  - 一つの項目に複数の文を詰め込まない
  - 理由や補足は、一段ネストした箇条書きにする
- 文体を統一する
  - README は「ですます」調
  - ADR・AGENTS.md は「だ・である」調
