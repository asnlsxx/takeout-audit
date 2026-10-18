# 首版验证与交接记录

日期：2026-10-18。状态：**alpha；本地检查、三平台 CI 与真实 Immich 3.2.0 隔离实例验证通过，尚无真实用户试用结论。**

## 已完成

| 检查 | 实际结果 |
|---|---|
| 来源与许可证 | 基于 dhcgn/immich-admin-cli 的 `5514efda26ba46afc3bb2d60a15ff723c3549992`；保留 MIT 与 NOTICE |
| API 规范 | 按仓库要求刷新并生成；规范信息版本 3.3.0-rc.0；在线运行仍限定正式 3.2.x |
| 完整测试 | **287 个通过事件（含子测试），0 失败**；原始记录位于本地 `dist/validation/tests.jsonl` |
| 静态检查 | go vet、go build、go fix 检查通过；无待格式化 Go 文件 |
| 生成一致性 | 连续两次生成的 README API 表、客户端和 Agent 技能内容保持一致 |
| 模糊测试 | 2 个进程、约 10 秒、1099 次输入执行，未发现失败；只覆盖一个有界 ZIP 入口，不能解释为无缺陷证明 |
| 合成基准 | 5000 个 512 字节成员、单次扫描约 73.9ms、累计分配约 11.1MiB；不是峰值内存或真实照片吞吐结果 |
| 依赖检查 | x/text 升至 v0.41.0；govulncheck v1.9.0 输出 `No vulnerabilities found.`；新增功能无新运行时依赖 |
| 发布配置 | actionlint v1.7.12 检查通过；临时服务脚本通过 Bash 语法检查 |
| 生命周期测试编译 | 可编译；本地因未开启隔离实例条件明确 **SKIP**，不计为真实服务通过 |
| 三平台原生运行 | Windows、Linux、macOS 的格式、构建、单元测试、生成文件与命令检查均通过 |
| 真实服务生命周期 | Immich **3.2.0**：matched=1、trashed=1、absent=1、extra source copies=1；受限查询 key；审计前后资产统计与回收站状态不变 |
| 真实测试清理 | 测试自身清除两项生成照片；作业结束删除全部临时容器、卷与网络 |
| Windows 实际使用 | 独立程序扫描两个演示 ZIP：6 条文件记录、4 次媒体出现、2 个内容组、2 个额外副本；源文件 hash 未改变；离线退出 2 符合约定 |

可核查的远端证据：[CI 39115445636](https://github.com/asnlsxx/takeout-audit/actions/runs/39115445636)，提交 `8c6b9644ed72aee2098de479412b59022a57e0cd`。真实生命周期日志明确记录 PASS，并非 SKIP。首次启动因测试脚本错误设置 `IMMICH_TELEMETRY_INCLUDE=none` 而失败；依据版本源码移除无效配置，并增加数据库就绪检查，随后通过。保留[首次失败记录](https://github.com/asnlsxx/takeout-audit/actions/runs/39114975747)，没有跳过失败步骤。后续发布还要求待发布提交自身的 CI 全部通过。

测试覆盖包括跨 ZIP 副本、同名不同内容、ZIP 内重名成员、非 UTF-8 路径原字节、JSON/未知格式、符号链接、CRC 错误、限制与取消、CSV 公式字符、哈希碰撞、缺失账户/版本字段、权限失败、接口漏项/多项/重复项、缺失回收站标记、分批与重试、离线零联网、输入未改动、禁止越界请求/重定向、重复/交叠输入、独立命令不暴露写操作。

## 尚未验证与稳定版门槛

1. 真实多包 Takeout 的性能、格式覆盖、2—3 名目标用户试用；无用户采用或稳定性承诺。
2. 当前实际服务器测试只覆盖 3.2.0；其他 3.2.x 补丁版本未逐个验证。元数据、相册、外部库、远端磁盘完整性与导出包是否齐全均不在首版核验范围。
3. 过期或权限不足 key 已有模拟接口测试，仍需更广泛的真实服务回归；大型导出不能以小文件合成基准代替。
4. 源码发布在[用户自己的仓库](https://github.com/asnlsxx/takeout-audit)，以 alpha 安装包收集受控试用反馈，不向上游社区自动发帖，也不称为稳定版本。

## 交接入口

- `START-HERE.zh-CN.md`：独立命令使用说明。
- `docs/takeout-audit.md`：输入输出、语义和边界。
- `docs/implementation-plan.md`：交付范围及停止扩展的条件。
- `internal/takeoutaudit/`：无网络扫描、分组和报告。
- `internal/workflows/takeoutaudit.go` 与 `internal/client/audit.go`：查询与只读请求约束。
- `cmd/takeout-audit/`：独立入口；保留的上游入口也支持 `cw audit-takeout`。
- `.github/workflows/ci.yml` 与 `test/immich/`：三平台检查及临时服务生命周期验证。
- `dist/validation/`：少量本地测试日志，已忽略，不含私有照片或凭据。

后续无需重新选题或研究整套市场。优先真实用户小范围试用；出现任何错误状态误判为匹配、审计改变资产、输出漏记已枚举输入的情况，必须修复后再发布。若真实用户用现有检查命令已足够，停止扩展产品，优先向上游贡献有用的测试与对账能力。
