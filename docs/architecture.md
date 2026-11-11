# 架构

Toki 是一个纯 Go 的单二进制应用：MyGo 原生 UI，无前端运行时、无 cgo，代码在 `app/`。

## 目录

```
app/
├── app.go             app 状态、视图骨架、工具栏/侧栏/网格/时间轴/状态栏、偏好读写、main()
├── model.go           Photo / Album / EditState / Library（JSON 库、原子保存、互斥）
├── library_index.go   库的派生索引：排序快照、按路径查找、侧栏计数（按变更重建）
├── openfile.go        从 Finder / 文件管理器打开图片，库外照片按单张查看
├── scan.go            目录遍历、增量判断、内容指纹去重、清理失效项
├── exif.go            EXIF 解析（TIFF IFD/GPS）+ XMP 写入（JPEG 内嵌 / 边车）
├── imageops.go        解码（JPEG/PNG/GIF/BMP/WebP/TIFF）、方向校正、缩放（面积平均）、
│                      旋转/翻转/拉直/裁剪、JPEG 写出（透明铺白）、水印、指纹
├── anim.go            动画 GIF：合成帧、按延时播放、长动画跳帧限内存
├── confetti.go        收藏时的彩纸：自绘粒子、弹道与淡出
├── thumbs.go          缩略图：磁盘缓存 + 内存 LRU（GPU 位图）+ 后台工作池
├── actions.go         菜单、导入/刷新、星标、移出库、写元数据、全屏、文件监视、Finder 定位
├── export.go          导出管线与命名
├── dialogs.go         导出 / 设置 / 相册 / 标签 四个模态
├── ui_*.go            标题栏、侧栏、照片墙、检查器、查看器、控件与主题
├── *_test.go          单元测试 + 视图测试 + 截图渲染
├── mygo.json          应用标识、版本与打包配置
└── resources/icon.png 图标
```

## 数据模型

```go
Photo{Path,Name,Root,Size,ModUnix,TakenUnix,Width,Height,Orientation,
      Stars,Caption,Tags,Albums,Edit,Undo,Redo,
      Make,Model,ISO,FNumber,ExposureTime,FocalLength,HasGPS,Lat,Lon}

EditState{Rotate,FlipH,FlipV,Angle,
          HasCrop,CropX,CropY,CropW,CropH}   // 归一化裁剪矩形 + 拉直角度

Album{ID,Name,Starred,Created}
Library{Folders,Photos,Albums,Tags,Prints}   // Prints: 指纹 → 路径，用于去重
```

- **持久化**：`library.json` 整体序列化，写入用「临时文件 + rename」保证原子性；`settings.json` 独立。
  单用户桌面库（万级照片）足够；换 SQLite 是后续可选升级（见 `decisions.md`）。
- **并发**：`Library.mu` 保护所有读写；扫描/导出/元数据写入在工作协程，结果通过
  `mygo.RunOnMain` + `win.Invalidate()` 回到主线程重绘（`winReady` 闸门防止首帧前通知窗口）。
- **派生数据**：`libraryData` 按原子 revision 缓存排序快照与侧栏计数；`flowModel` 按照片/宽度/排列缓存
  布局，并用按 y 排序的索引做二分查找；时间轴、胶片条与侧栏都只构建可见部分。一帧的复杂度与照片总数无关。
- **原文件只读**：编辑、星级、标签都只存库里；「移出库」不改磁盘文件。

## 缩略图

1. 键 = `sha1(路径 + mtime + 请求尺寸 + 编辑 JSON)`，所以**编辑一改，缩略图自动失效重生**。
2. 命中内存 LRU（上限 400 张，超出 `ReleasePixels` 释放 CPU 像素）→ 直接返回。
3. 否则读磁盘 `thumbs/<key>.jpg`，`ui.DecodeBitmap` 后入 LRU。
4. 都没有：投递到后台工作协程（队列 512，满了就放弃本次，下一帧再要）。
   磁盘上已有的缩略图只由协程读取、在主线程解码；缺失的才做 `解码 → 方向校正 → 应用编辑 → 按尺寸缩放 → 写 JPEG`。
   主线程从不做文件 I/O 或 JPEG 解码，滚动不会因为缩略图而停顿。
5. 网格只构建可见单元格，所以 1 万张照片与 10 张的首帧开销相同。

## 动画 GIF

`anim.go` 用 `gif.DecodeAll` 合成每一帧（含 disposal），按每帧延时播放；为限制内存，
帧数上限 120、总像素约 40 MP、边长上限 1200，超出的长动画按步长跳帧并合并延时。

## 渲染与编辑

`renderEdit(img, st)` 顺序固定：**旋转 90° 步进 → 水平/垂直翻转 → 拉直（双线性，扩边）→ 裁剪（归一化矩形）**。
编辑面板把改动写在草稿（`app.draft`）上，预览键用草稿，「应用」才落地成一个可撤销步骤。

## 监视

每个受监视文件夹一个 `mygo.WatchFiles`（递归、600 ms 去抖、忽略点目录与系统目录）。
收到图片类事件后**只重扫该文件夹**（增量判断，未变文件直接跳过），入库后刷新视图并给出状态提示。

## 元数据

- 读：解析 JPEG 的 APP1 段（`II`/`MM` TIFF、IFD0 → ExifIFD → GPS IFD），不引入完整 EXIF 库。
- 写：构造最小 XMP 包（`dc:title` / `dc:subject` / `xmp:Rating`），若已有
  `http://ns.adobe.com/xap/1.0/` 段则原地替换，否则插在 SOI 之后；重写文件用临时文件 + rename。
  非 JPEG 写 `<同名>.xmp` 边车。

## 测试

`go test ./...` 覆盖：扫描/增量/去重/清理、库往返、相册/标签/筛选、EXIF 解析（合成 TIFF）、
XMP 写入与替换、旋转/翻转/裁剪/拉直/缩放、导出（尺寸档位、应用的编辑）、撤销重做、
文件监视（真实文件系统事件）、三种视图的无窗口渲染（`ui.NewTester`）+ 文案断言。

`TestPreview` 是手工工具：把真实库渲染成 PNG 便于目视检查（`TOKI_PREVIEW_DATA` / `TOKI_PREVIEW_OUT`）。

性能回归在 `performance_test.go`：5000 张照片的网格 / 时间轴 / 胶片条只应构建可见元素，侧栏索引只应重建一次。
