# 界面

窗口是一个原生三栏工作台：自绘标题栏（保留系统红绿灯）、侧边栏、照片墙 / 查看器、检查器。
深色与浅色两套 Token，默认跟随系统外观。截图见 `docs/screenshots/`，项目主页见仓库的 GitHub Pages。

## 运行与预览

```bash
cd app
go tool mygo dev          # 开发模式，改 .go 自动重启
go test ./...             # 无窗口测试（库、编辑、导出、视图、键盘）
go tool mygo build        # 打包 .app / .dmg（见「构建」）
```

想把示例照片导进一个空库：

```bash
TOKI_SEED="$PWD/testdata/demo-photos" go tool mygo dev
```

截图（不开窗口，把视图渲染成 PNG；需要 macOS 的 `sips`）：

```bash
python3 bin/preview_seed.py app/.preview
cd app
TOKI_PREVIEW_DATA="$PWD/.preview" TOKI_PREVIEW_OUT=../docs/screenshots \
  go test -run 'TestPreview' -v
```

`TestPreviewStates` 另外渲染选中、命令面板、新建相册、批量选择与收藏彩纸。`.preview/` 只在本地存在。

## 布局与控件

| 区域 | 内容 | 文件 |
| --- | --- | --- |
| 标题栏（48px） | 品牌标记与「時」、所在位置、搜索框（⌘K）、外观、导入、信息面板、设置；整条可拖动窗口 | `app/ui_shell.go` |
| 侧边栏（240px） | 全部照片 / 最近添加 / 收藏 / 星标照片 / 已编辑；我的相册（六色圆点 + 计数）、标签、文件夹、年份；底部是库占用 | `app/ui_shell.go` |
| 视图头 | eyebrow、标题、副标题、新建相册 / 导入照片；按图库、相册、标签、文件夹、年份、搜索变化 | `app/ui_library.go` |
| 筛选与排列 | 全部 / 收藏 / 已编辑 / 四星以上；排序、网格 / 瀑布流 / 时间轴、三档缩略图、结果数 | `app/ui_library.go` |
| 照片墙 | 网格按日分组，瀑布流按最矮列；布局按变更缓存，只构建可见卡片；时间轴只构建可见行；卡片有心形、选择勾与名称条 | `app/ui_library.go` |
| 日期滑块 | 瀑布流右缘；手柄可拖动、滚轮可滚动、点轨道跳到最近日期，悬停提示写出日期；手柄中心与刻度同步 | `app/ui_library.go` |
| 检查器（272px） | 预览、星级与收藏、照片信息、标签、所属相册、操作；多选为批量面板 | `app/ui_inspector.go` |
| 查看器 | 深色舞台、胶片条、序号、星级、收藏、导出、编辑、幻灯片；滚轮缩放、拖动平移；胶片条只构建可见缩略图，并跟随当前照片滚动 | `app/ui_viewer.go` |
| 命令面板（⌘⇧K） | 照片、相册、标签、文件夹统一搜索 | `app/ui_shell.go` |
| 对话框与提示 | 导出 / 设置 / 相册 / 标签四个模态，`Context.Toast` | `app/dialogs.go` |

视觉按 4px 栅格、6px 控件圆角、1px 描边与单一强调色绘制；照片舞台上没有阴影、滤镜或渐变。

## Token

`app/theme.go` 把设计 Token 映射为 `ui.Theme`：冷色墨蓝中性色、一个强调色用于选中与焦点，
收藏粉、评分黄、成功绿、危险红各只用在一个语义上。深色与浅色共用同一套间距与字号。

## 交互约定

- **收藏**：卡片、检查器、查看器与 `F` 键都能收藏；心形只在卡片悬停、选中或已收藏时出现。
- **多选**：卡片左上角的勾选框是真正的多选开关，不取消其他已选照片。
- **标签 / 文件夹**：侧栏标签右键可删除（从所有照片与词表中移除）；文件夹右键可移除，确认框写明只移除资料库记录、不删除本地文件，并可勾「以后不再提醒」（设置里可重新打开）。移除后已收藏的照片留在资料库。
- **查看**：双击卡片或右键「全屏查看」进入查看器；`←`/`→` 切换，`Esc` 关闭。滚轮缩放，高分辨率鼠标的精确滚动同样有效。
- **从系统打开**：同时注册扩展名与系统类型（`fileAssociations` + `LSItemContentTypes`）；把图片的「打开方式」设为 Toki 后，
  Finder / 文件管理器双击即进入查看器。库外照片以所在文件夹为临时片夹：胶片条与 `←`/`→` 切换该文件夹里的图片，
  不写入图库；收藏、打星、导出、编辑这些只在库里成立的动作用到它时，才把这一张加入资料库。
- **编辑**：非破坏性，草稿实时预览，「应用」写入撤销栈（每张上限 50 步）。
- **缩略图尺寸**：小 / 中 / 大切换时，卡片宽度用约 180ms 的缓出动画过渡，不做瞬间跳变。
- **时间轴**：工具栏的排列按钮与「显示 > 时间轴」（⌘3）；侧栏不重复这一项。

## 无障碍与键盘

所有控件有可读的 name/tooltip；卡片保持按钮语义（Enter / Space 选中）。
快捷键见 `README.md` 的键盘表。

## 验证

```bash
cd app
gofmt -l .
go vet ./...
go tool mygo vet .
CGO_ENABLED=0 go test ./...
```

截图按外观与状态分组放在 `docs/screenshots/`，浅色的带 `-light` 后缀：

- `grid.png` / `grid-narrow.png` / `grid-light.png` / `grid-narrow-light.png` — 网格，深色与浅色、宽与窄
- `masonry.png` / `masonry-light.png` / `masonry-scrolled.png` / `masonry-scrolled-light.png` — 瀑布流与日期滑块
- `timeline.png`、`oneup.png`、`edit.png`（及各自的 `-light`）— 时间轴、单张查看、编辑面板
- `lightbox.png` / `lightbox-zoom.png` / `lightbox-half.png` — 查看器与缩放；舞台在两种外观下都是深色，所以只有一套
- `command.png`、`multi.png`、`new-album.png`、`confetti.png`、`export.png`、`search.png`（及各自的 `-light`）— 命令面板、多选、新建相册、收藏彩纸、导出、搜索

`make preview` 先渲染深色一套，再用 `TOKI_PREVIEW_THEME=light` 渲染浅色一套（输出 `*-light.png`）。不传这个变量时，预览用库自己的外观设置。
