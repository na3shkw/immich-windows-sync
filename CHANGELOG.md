# Changelog

## [v0.1.2](https://github.com/na3shkw/immich-windows-sync/compare/v0.1.1...v0.1.2) - 2026-10-29

### Other Changes
- Bump github.com/labstack/echo/v4 from 4.15.3 to 4.16.1 by @dependabot[bot] in https://github.com/na3shkw/immich-windows-sync/pull/18
- Bump vite from 8.3.1 to 8.3.4 in /frontend by @dependabot[bot] in https://github.com/na3shkw/immich-windows-sync/pull/17
- Bump postcss-selector-parser from 6.1.4 to 6.1.5 in /frontend by @dependabot[bot] in https://github.com/na3shkw/immich-windows-sync/pull/21
- Bump source-map-js from 1.2.1 to 1.2.2 in /frontend by @dependabot[bot] in https://github.com/na3shkw/immich-windows-sync/pull/15
- Bump github.com/wailsapp/wails/v2 from 2.11.0 to 2.13.0 by @dependabot[bot] in https://github.com/na3shkw/immich-windows-sync/pull/20

## [v0.1.1](https://github.com/na3shkw/immich-windows-sync/compare/v0.1.0...v0.1.1) - 2026-09-27

### Bug Fixes 🐛
- updated_atをUTCで保存するよう修正 by @na3shkw in https://github.com/na3shkw/immich-windows-sync/pull/2
- スタートアップ起動時はウィンドウを表示せずタスクトレイに常駐する by @na3shkw in https://github.com/na3shkw/immich-windows-sync/pull/14
### Other Changes
- Bump golang.org/x/crypto from 0.33.0 to 0.52.0 by @dependabot[bot] in https://github.com/na3shkw/immich-windows-sync/pull/3
- Bump github.com/labstack/echo/v4 from 4.13.3 to 4.15.3 by @dependabot[bot] in https://github.com/na3shkw/immich-windows-sync/pull/4
- npm 依存パッケージの一括アップデート by @na3shkw in https://github.com/na3shkw/immich-windows-sync/pull/13
- Bump golang.org/x/net from 0.54.0 to 0.55.0 by @dependabot[bot] in https://github.com/na3shkw/immich-windows-sync/pull/7

## [v0.1.0](https://github.com/na3shkw/immich-windows-sync/commits/v0.1.0) - 2026-09-26

### New Features 🎉

- 接続設定・フォルダ管理・同期状況・スタートアップ設定の各画面
- Windows → Immich の一方向同期（自動・手動）
- 同期対象フォルダ・除外フォルダの設定
- Windows ログイン時のスタートアップ登録

### Documentation 📝

- README や AGENTS.md、ADR を追加

### Other Changes

- CI やリリースに利用するワークフローの整備
