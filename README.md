# 時 · Toki

A local-first photo library, built as a native desktop app in pure Go.
纯 Go 的本地照片库：原生三栏界面、非破坏性编辑、完全离线。

[![CI](https://github.com/ilinmu/Toki/actions/workflows/ci.yml/badge.svg)](https://github.com/ilinmu/Toki/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/ilinmu/Toki?display_name=tag)](https://github.com/ilinmu/Toki/releases)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

当前版本：**v0.1.5** · [下载](https://github.com/ilinmu/Toki/releases/tag/v0.1.5)

![网格视图](docs/screenshots/grid.png)

## 特性

- **照片墙**：网格按日期分组、瀑布流按最矮列、时间轴；布局、侧栏计数按变更缓存，滚动时只构建可见卡片 / 行 / 胶片条，缩略图在后台加载，万级照片也能立刻打开。
- **查看器**：深色舞台、胶片条、滚轮（含高分辨率鼠标）以光标为中心缩放（10%–800%）、拖动平移、幻灯片。
- **Finder 双击打开**：已注册 JPEG / PNG / GIF / BMP / WebP / TIFF 的扩展名与系统类型（UTI），把图片的「打开方式」设为 Toki 并「始终以此方式打开」后，同格式的照片都从这里打开。库外的照片以它所在的文件夹作为临时片夹：胶片条与左右键切换的是这个文件夹里的图片，不写入图库；只有收藏它，这一张才加入资料库。
- **整理**：星标、收藏、标签（右键可删除）、相册、按文件夹与年份浏览、组合搜索（文本 × 星级 × 来源 × 排序 × 筛选）。
- **移除文件夹**：右键文件夹即可从资料库移除；会先确认「不会删除本地文件」（可勾「以后不再提醒」关掉，也能在设置里打开），之后可以重新导入。已收藏的照片会留在资料库里，不会跟着文件夹消失。
- **非破坏性编辑**：裁剪（比例预设）、拉直、旋转翻转、曝光 / 对比度 / 饱和度 / 色温 / 暗角，逐张撤销重做。
  缩略图、预览与导出共用同一份渲染结果。
- **导出**：8 档尺寸、4 档质量、文件名加序号、水印。
- **元数据**：读 EXIF（相机、ISO、光圈、曝光、焦距、GPS、拍摄时间）；写 XMP（JPEG 内嵌，其它格式写边车）。
- **监视文件夹**：新照片自动出现；原文件永远只读；导入后在后台预生成缩略图与小预览，减少首次浏览的「载入中」。
- **格式**：JPEG、PNG、GIF（动画会播放）、BMP、WebP、TIFF。
- **纯本地**：不联网、无账号、无遥测；库保存在 `~/Library/Application Support/Toki/`（Linux 为 `~/.config/Toki/`）。

## 下载

macOS、Linux、Windows 的安装包在 [Releases](https://github.com/ilinmu/Toki/releases)：

| 平台    | 产物                                      |
| ------- | ----------------------------------------- |
| macOS   | `toki-<版本>-universal.dmg`               |
| Linux   | `toki-<版本>-linux-<架构>.tar.gz`、`.deb` |
| Windows | `toki-setup-<版本>-<架构>.exe`            |

macOS 构建为 ad-hoc 签名（未公证），首次打开需要在「系统设置 › 隐私与安全性」里允许。

## 从源码构建

需要 Go 1.27+；不需要 Node、cgo 或 Xcode（打包 dmg 需要 macOS）。

```sh
cd app
go tool mygo dev      # 开发模式
go test ./...         # 无窗口测试
go tool mygo build    # 打包到 build/
```

想用示例照片试：`TOKI_SEED="$PWD/testdata/demo-photos" go tool mygo dev`。

## 技术栈

| 层 | 选择 | 说明 |
| --- | --- | --- |
| 语言 | Go 1.27+ | 单二进制，`CGO_ENABLED=0`，没有 Electron / Node / 前端运行时 |
| UI 与窗口 | [MyGo 0.8](https://github.com/egoist/mygo) | 原生控件、自绘标题栏、菜单、文件监视与打包（`.app` / `.dmg` / `.deb` / `.exe`），纯 Go 无 cgo |
| 图像 | `image/jpeg` / `image/png` / `image/gif`，`golang.org/x/image/{bmp,webp,tiff}` | 解码、方向校正、面积平均缩放、编辑渲染；动画 GIF 按帧合成与播放 |
| 数据 | JSON + 原子 rename | `library.json`、`settings.json`，无数据库；缩略图为磁盘 JPEG + 内存 LRU |
| 元数据 | 自写 EXIF / XMP | 解析 JPEG APP1，写 XMP（JPEG 内嵌，其它格式 `.xmp` 边车） |
| 构建与发布 | GitHub Actions + `go tool mygo build` | 三平台测试、密钥扫描、CodeQL、打标签自动发布 Release |
| 项目主页 | GitHub Pages | `docs/index.html`，单文件静态页 |
| 许可 | MIT | 见 [`LICENSE`](LICENSE) |

## 开发

```sh
make test    # go test ./...（在 app/）
make lint    # gofmt、go vet、mygo vet、staticcheck
make build   # 当前平台的安装包
```

更多内容：

- [`docs/architecture.md`](docs/architecture.md) — 模块、数据模型、缩略图、编辑与监视
- [`docs/ui.md`](docs/ui.md) — 界面结构、Token 与交互约定，截图说明
- [`docs/decisions.md`](docs/decisions.md) — 工程取舍
- [`docs/todo.md`](docs/todo.md) — 明确推迟的功能
- [`CONTRIBUTING.md`](CONTRIBUTING.md) — 提交流程与代码风格
- [`SECURITY.md`](SECURITY.md) — 安全问题怎么报

## 键盘

| 键           | 作用                               |
| ------------ | ---------------------------------- |
| ⌘K           | 聚焦搜索框（输入即筛选，Esc 清空） |
| ⌘⇧K          | 命令面板                           |
| ⇧⌘L          | 深色 / 浅色外观                    |
| ⇧⌘I          | 显示或隐藏照片信息                 |
| ⌘1 / ⌘2 / ⌘3 | 网格 / 瀑布流 / 时间轴             |
| ⌘O / ⌘R / ⌘E | 导入文件夹 / 刷新库 / 导出         |
| ⇧⌘N          | 新建相册                           |
| ⌘A           | 全选                               |
| F            | 收藏当前照片                       |
| 1–5          | 查看器里打星                       |
| ← / → / Esc  | 查看器里切换 / 关闭                |

## 截图

|                                         |                                          |
| --------------------------------------- | ---------------------------------------- |
| ![瀑布流](docs/screenshots/masonry.png) | ![查看器](docs/screenshots/lightbox.png) |
| ![单张查看](docs/screenshots/oneup.png) | ![编辑](docs/screenshots/edit.png)       |

更多在 [`docs/screenshots/`](docs/screenshots/)。

## 许可

[MIT](LICENSE)。
