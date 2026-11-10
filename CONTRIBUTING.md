# 贡献指南

感谢参与。这个项目是纯 Go 的桌面应用（MyGo 原生 UI），没有前端构建步骤。

## 环境

- Go 1.27 或更新（以 `app/go.mod` 为准）
- 不需要 Node、cgo 或 Xcode；打包 macOS `.dmg` 需要 macOS

## 本地开发

```sh
cd app
go tool mygo dev      # 开发模式
go test ./...         # 无窗口测试
go tool mygo build    # 当前平台的安装包
```

仓库根目录也有 `make test` / `make lint` / `make build`。

## 提交前

```sh
cd app
gofmt -l .            # 应当没有输出
go vet ./...
go tool mygo vet .
CGO_ENABLED=0 go test ./...
```

界面改动请附一张用 `TestPreview` 渲染的截图（见 `docs/ui.md`）。

## 提交与 PR

- 一个 PR 做一件事；合并前 CI 必须全绿。
- 提交信息用祈使句，首行不超过 72 字符，例如 `fix: keep the date rail reachable between ticks`。
  常用前缀：`feat:`、`fix:`、`docs:`、`test:`、`refactor:`、`chore:`。
- PR 描述写清：改了什么、为什么、怎么验证（命令或截图）。
- 不要提交构建产物、`.preview/`、个人库数据或任何密钥；`gitleaks` 会在 CI 里扫描。

## 代码风格

- 遵循 Go 惯例：错误要处理，导出标识符要有注释；注释解释「为什么」，不复述代码。
- 界面状态放在 `app` 上，不把 `ui.Element` / `*ui.Context` 存进长期状态（见 MyGo 的构建约定）。
- 新增交互尽量补一个 `ui.NewTester` 回归测试，断言用户看到的结果，而不是实现细节。
- 新增字符串面向用户时用中文；代码标识符与注释用英文。

## 报告问题

用 issue 模板，附上系统版本、复现步骤、期望与实际结果。安全问题请走 `SECURITY.md`，不要公开细节。
