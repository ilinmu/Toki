# 時 · Toki — 应用

Toki 的照片库应用本体：纯 Go + MyGo 原生 UI，无 cgo、无前端运行时。
面向用户的说明、下载与截图在仓库根目录的 [`README.md`](../README.md)。

## 运行

需要 Go 1.27+。

```sh
go tool mygo dev      # 开发模式，改 .go 自动重启
go test ./...         # 无窗口测试（库、编辑、导出、视图、键盘）
go tool mygo build    # 打包：build/darwin-arm64/{Toki.app, *.dmg}
go tool mygo vet .    # 检查存储的构建值与协程捕获
```

首次启动：点「导入文件夹」，选一个照片文件夹即可（只索引，不移动、不修改原文件）。

用示例照片试：

```sh
TOKI_SEED="$PWD/testdata/demo-photos" go tool mygo dev   # 首次启动即导入该文件夹
```

`TOKI_SEED` 只在库为空时生效；旧名 `LIBRA_SEED` 仍然认。

## 代码地图

| 文件 | 内容 |
| --- | --- |
| `app.go` | app 状态、窗口骨架、偏好读写、`main()` |
| `model.go` | `Photo` / `Album` / `EditState` / `Library`，JSON 库与原子保存 |
| `scan.go` / `thumbs.go` | 目录扫描、指纹去重、缩略图缓存与后台工作池 |
| `imageops.go` / `anim.go` | 解码、方向、缩放、编辑渲染、JPEG 写出；动画 GIF |
| `exif.go` / `export.go` | EXIF 读、XMP 写、导出管线 |
| `ui_*.go` | 标题栏、侧栏、照片墙、检查器、查看器、控件与主题 |
| `dialogs.go` / `actions.go` | 模态与菜单、文件监视、Finder 定位 |
| `confetti.go` | 收藏时的彩纸 |

架构与取舍见 [`../docs/architecture.md`](../docs/architecture.md)、[`../docs/decisions.md`](../docs/decisions.md)。

## 数据位置

- macOS：`~/Library/Application Support/Toki/`
- Linux：`~/.config/Toki/`（`os.UserConfigDir()`）
- Windows：`%AppData%\Toki\`

里面是 `library.json`（照片、相册、标签、编辑与撤销栈）、`settings.json`（偏好）、`thumbs/`（缩略图缓存，可安全删除）。
旧版本把库放在 `Libra/` 时，应用会继续读那一份，不另建空库。

## 截图

视图可以不开窗口渲染成 PNG（`ui.Tester`）。需要 macOS 的 `sips` 生成缩略图：

```sh
cd ..
python3 bin/preview_seed.py app/.preview
cd app
TOKI_PREVIEW_DATA="$PWD/.preview" TOKI_PREVIEW_OUT=../docs/screenshots \
  go test -run 'TestPreview|TestPreviewStates' -v
```

`.preview/` 只在本地存在，已在 `.gitignore` 里。旧名 `LIBRA_PREVIEW_*` 仍然认。

## 说明

应用不含任何第三方的代码、图标、位图或本地化文案；图标与示例照片都随仓库提供。
