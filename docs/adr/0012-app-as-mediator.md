# 0012. Syncer と Watcher を独立させ、App を仲介役にする

- ステータス: 承認済み
- 日付: 2026-06-20

## 背景

- ファイルの変更を検知する処理（Watcher）と、アップロードする処理（Syncer）がある
- 両者を直接つなぐと、互いの内部に依存してテストもしにくくなる

## 決定

### 責務の分担

- **`Watcher`（`internal/watcher`）はイベントを流すだけにする**
  - fsnotify のイベントを自前の `Event` 型に変換する
  - 変換したイベントをチャネル（`Events` / `NewDirectories`）に流す
  - fsnotify への依存は watcher パッケージの中に閉じる
- **`Syncer`（`internal/syncer`）は渡されたファイルをアップロードするだけにする**
  - `SyncAssets` は同期関数として書く
  - 非同期にするかどうかは呼び出し側が決める
- **`app.go` の `App` が仲介役（mediator）になる**
  - `App` が Watcher と Syncer の両方を保持する
  - Watcher のチャネルを受けて、Syncer を呼ぶ

### 実装上のルール

- **監視・スキャン対象のフォルダ（`targetDirs`）は引数で渡す**
  - struct のフィールドにはしない
  - 設定変更のたびに変わる値のため
- **チャネルを読み出す goroutine は、アプリ起動時に 1 回だけ立ち上げる**
  - Watcher は設定変更時に Stop → Start で再起動される
  - ただしチャネルは、Watcher の生存期間を通じて 1 つだけ
  - `StartWatcher` の中で起動すると、goroutine が増え続ける
- **同じパスへのイベントは、パスごとにデバウンス（500ms）してから同期する**
  - ファイルのコピーなどで、Create / Write が短時間に何度も発火するため
  - 実装は `internal/debounce`
- **`StartWatcher` は監視を始める前に全体スキャン（`SyncNow` 相当）を行う**
  - スキャンとアップロードは goroutine で非同期に実行する
  - フロントエンドをブロックしないため

## 結果

- Watcher と Syncer を、それぞれ単体でテストできる
- つなぎ方の変更は `app.go` だけで済む

> 補足: パッケージ名は、標準ライブラリの `sync` と衝突しないよう `sync` から `syncer` に変更した（2026-07-04）。
