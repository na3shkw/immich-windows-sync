# 0015. dev ビルドと本番ビルドの切り替えにビルドタグを使う

- ステータス: 承認済み
- 日付: 2026-09-23

## 背景

- `wails dev` で開発中のアプリが、本番アプリの次のものを書き換えてしまうと困る
  - 設定・DB・同期ログ
  - スタートアップ登録

### 以前の方式（2026-06-27〜2026-09-23）

- スタートアップ登録の実装時は、dev ビルドかどうかを `runtime.Environment(ctx).BuildType` で判定していた
- `main.go` で `--dev` フラグを受け取る案と比べて、この方式を選んだ
  - 手動で渡さなくても自動で判定できる
  - 登録処理は UI のトグルから呼ばれるため、その時点では `ctx` を使えた
- `startup` パッケージには Wails 固有の型を持ち込まなかった
  - `app.go` が `BuildType == "dev"` を評価して `isDev bool` に変換していた
  - それを `startup.NewStartup(isDev)` に渡していた
  - `startup` パッケージがキー名に `-dev` を付けていた

### 問題

- その後、設定・DB・同期ログも dev と本番で分けたくなった
- しかし `runtime.Environment` には Wails の `ctx` が必要
- そのため、`startup()` の中で `LoadConfig()` を呼ぶより前には判定できない

## 決定

- `wails dev` が `-tags dev` でビルドすることを利用する
- `internal/appenv` に、同じ名前の定数（`appDirName`・`RegistryKey`）を別々の値で定義する
  - `appenv_dev.go`（`//go:build dev`）: `immich-sync-dev` / `ImmichWindowsSync-dev`
  - `appenv_prod.go`（`//go:build !dev`）: `immich-sync` / `ImmichWindowsSync`
- データフォルダのパスは、必ず `appenv.AppDir()` を通して取得する

## 結果

- dev ビルドは `%APPDATA%\immich-sync-dev\` を使い、本番のデータやスタートアップ登録に触れない
- 判定はコンパイル時に決まるため、実行時の分岐や `ctx` は不要になる
- `startup` パッケージは `isDev` を受け取らず、`appenv.RegistryKey` を直接使う
- dev 版と本番版は別の名前で登録されるため、両方を同時にスタートアップ登録できる
- 当初の分離の目的は記録に残っていない
  - 考えられる目的は「両方を同時に登録したい」か「開発中に本番の登録を誤って書き換えない」
  - 現在の実装はどちらも満たしている
