# 0011. Immich API クライアントの設計

- ステータス: 承認済み
- 日付: 2026-02-21

## 背景

- Immich へのアップロードには `POST /api/assets`（multipart/form-data）を使う
- 認証は `x-api-key` ヘッダーで行う
- 数 GB 級の動画も扱う

## 決定

- アップロードするデータは `io.Reader` で受け取る
- multipart のフィールドは次のようにする
  - `deviceId`: アプリ名の固定文字列（`immich-windows-sync`）
  - `deviceAssetId`: ファイルパス
  - `fileCreatedAt`: ファイルの更新日時（`ModTime`）
- レスポンスの重複判定は `bool` にせず、Immich が返す `status` 文字列のまま保持する
  - 値は `created` / `duplicate` など

## 検討した選択肢

- **`[]byte` で受け取る**
  - ファイル全体がメモリに載るため、大きな動画に向かない
- **`*os.File` で受け取る**
  - テストで差し替えにくい

## 結果

- メモリ使用量を抑えられる
- テストでは `strings.NewReader` などに差し替えられる
- EXIF の撮影日時は使っていない
  - コピーなどで更新日時が変わったファイルは、Immich 上の日付がずれる場合がある
  - EXIF の解析は今後の課題
