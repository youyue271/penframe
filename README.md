# Omni-Pentest Orchestrator (Safe MVP Skeleton)

这个目录现在是一个偏核心编排层的 Go 骨架，目标是把你定义的四层架构先落到一个可演进的 MVP 上：

- `Tool Manager`: 工具目录、变量定义、命令模板
- `Parser Engine`: YAML 驱动的正则解析，统一沉淀到 `assets` 结构
- `Workflow Engine`: 节点、边、条件判断、变量上下文
- `Executor`: 提供 `mock` 与 `local` 执行器（`local` 在 WSL 中可自动通过 `powershell.exe` 调起 `.exe`）
- `Storage`: 先给出内存存储占位，后续可以替换 PostgreSQL / Redis

## 安全边界

这里没有实现下列能力：

- WebShell / VShell / C2 适配
- 内网横向或反连 Agent
- 自动化利用链或漏洞攻击模块
- 任意命令下发执行

这些部分会直接提升攻击能力，不在这个骨架里落地。当前实现只覆盖安全边界内的编排核心，方便你先把模型、数据流和接口定下来。

## 目录结构

- `cmd/opo`: CLI 入口
- `cmd/portal`: Web UI 入口
- `internal/domain`: 领域模型
- `internal/config`: YAML 加载
- `internal/tooling`: 工具注册表
- `internal/parser`: 结果解析引擎
- `internal/runtime`: 模板渲染
- `internal/executor`: 执行器接口与 `mock` / `local` 执行器
- `internal/workflow`: 工作流执行与条件求值
- `internal/portal`: 嵌入式 Web UI 与 API
- `internal/storage`: 内存存储占位
- `examples/mvp`: 示例工具、流程、解析规则、fixture 输出

## 当前工作流

示例流程如下：

1. `discover` 读取 `fixtures/discovery.txt`
2. `service_discovery` 解析规则把 HTTP / MSSQL 资产写入 `assets`
3. 边条件 `len(assets.services.http) > 0 && results.discover.record_count > 0` 成立
4. `enrich_http` 读取 `fixtures/enrich.txt`
5. `asset_enrichment` 继续补充 `assets.http_metadata`

## 运行方式

当前已经按下面这组命令完成构建验证：

```bash
go mod tidy
go test ./...
go run ./cmd/opo -tools examples/mvp/tools.yaml -workflow examples/mvp/workflow.yaml
go run ./cmd/opo -tools examples/mvp/tools.yaml -workflow examples/live/workflow.yaml
go run ./cmd/portal -tools examples/mvp/tools.yaml -workflow examples/mvp/workflow.yaml
```

CLI 预期输出是一个 JSON 运行摘要，包含：

- `vars`
- `assets`
- `node_results`
- `execution_order`

Portal 默认监听 `http://localhost:8080`，提供：

- 工作流拓扑可视化
- Demo workflow 执行按钮
- 配置热重载按钮
- 最近运行记录
- 资产与节点执行明细
- 只读外部工具文件浏览
- 浏览器本地持久化的工具配置草稿
- 命令模板变量提示与输出文件模板字段
- 浏览器端解析规则预览
- 靶机地址输入与“开始任务（手动）”清单生成功能
- `GET /api/state`
- `POST /api/run`
- `GET /api/runs`
- `GET /api/runs/{id}`
- `POST /api/reload`
- `GET /api/tool-files`

当前运行摘要还会返回：

- `status`
- `error`
- `stats`
- 每个节点的 `executor`、`duration_millis`、`error`、`skip_reason`

工具工作台当前只覆盖安全范围内的元数据管理：

- 外部工具目录通过只读 API 暴露给前端浏览
- 页面里新增的工具配置保存在浏览器 `localStorage`
- 支持维护工具路径、命令模板、输出文件模板、解析规则和样例输出
- 可根据靶机地址生成手动执行清单（命令草稿与输出路径），不自动执行
- 解析预览只在浏览器中做正则匹配，不会调用本地执行器

执行器能力说明：

- `mock`: 读取 fixture 输出，适合离线回放
- `local`: 执行渲染后的 `command_template`
- 当命令首个可执行目标是 `.exe` 时，`local` 会自动用 `powershell.exe` 运行（适配 WSL）
- 每次运行会自动创建 `./output/<目标标识>/`，并注入 `vars.output_root` / `vars.output_dir` / `vars.output_dir_windows`
- `.exe` 命令参数里的输出路径建议使用 `{{ .vars.output_dir_windows }}`（Windows 路径）
- 节点 `metadata` 会记录 `launcher`、`command`、`exit_code`、`stderr` 等执行信息
- 若命令带有 `-o/-oN/-oA` 等输出参数，执行器会尝试回读输出文件并写入 `metadata.output_files`
- 执行器会在每次运行前清理本次声明的输出文件，避免历史扫描结果被追加混入

Portal 任务页支持直接输入靶机地址并启动任务：

- `POST /api/run` 支持 JSON 请求体：`{"target":"https://target.example:3000","vars":{...}}`
- `target` 会覆盖运行时 `vars.target / vars.target_host / vars.target_url`
- 可通过 `vars` 覆盖 live workflow 的工具链开关与参数（如 `run_nmap_full`、`run_nuclei_web`、`nmap_quick_args`、`fscan_args`、`nuclei_args`）
- WSL 通过 `powershell.exe` 跑 `nuclei.exe` 时建议在参数里加 `-sr`，避免偶发 DNS 解析失败（`no address found for host`）
- `POST /api/run` 可选 `timeout_seconds` 字段用于覆盖单次任务超时（默认 1800 秒，最小 30 秒，最大 86400 秒）

当前仍然没有实现：

- 实时监控或 tail 输出文件（需配合事件流）

## 下一步建议

如果你要继续按你的路线推进，建议按这个顺序扩展：

1. 把 `MiniExprEvaluator` 替换成正式的 `expr` 引擎
2. 给 `Executor` 增强命令审计、允许列表与节点级超时策略
3. 给 `RunSummary` 增加 PostgreSQL 持久化
4. 为 Portal 层补提交任意 workflow/tools 配置的 REST API
5. 增加实时事件流（WebSocket / SSE）和节点级日志视图
