# 0004. GUI に Wails v2 + React / TypeScript を採用する

- ステータス: 承認済み
- 日付: 2026-02-21

## 背景

- 次の画面を持つ設定 UI が必要
  - 接続設定
  - フォルダ管理
  - 同期ステータス
- バックエンドと UI は単一プロセス・単一実行ファイルにまとめたい

## 決定

- Wails v2（Go バックエンド + WebView2）を使う
- フロントエンドは React / TypeScript（`react-ts` テンプレート）で書く
- CSS は Tailwind CSS **v3** を使う
- `build/` ディレクトリは Git で管理する
  - アイコンやマニフェストなど、手で管理するファイルが入っているため
  - 成果物の `build/bin/` だけを除外する

## 検討した選択肢

- **CLI + 設定ファイルのみ**
  - 一般ユーザー向けの使用感にならない
- **getlantern/systray + 簡易ダイアログ**
  - 複数画面の設定 UI を作るには力不足
  - トレイ常駐には後に併用している（[0005](0005-startup-and-tray.md)）
- **Fyne**
  - Go だけで書ける
  - 既存の Web フロントエンドの経験を活かせない

## 結果

- Web フロントエンドの知見をそのまま使える
- Tailwind CSS は **v3 から上げない**
  - v4 は Wails v2 が使う Vite v3 と互換性がない
  - v4 では `npx tailwindcss init` も廃止された
- `frontend/wailsjs/` は Wails の自動生成物なので手で編集しない
