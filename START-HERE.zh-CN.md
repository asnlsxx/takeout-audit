# Takeout 对账工具：本地开发版

已实现一个可运行的首版，用来核对 Google Takeout 中的照片、视频与 Immich
账户内的内容记录。它保留每个源文件的去向，解释重复副本，并区分回收站、
未匹配、无法判断和读取错误。

当前为**alpha 实验版本**：本地模拟测试、三平台 CI、临时 Immich 3.2.0 联调均已通过。
在线模式暂限正式版 3.2.x。程序不会上传、删除或修复照片。

## 先试离线盘点

Windows 开发包中的可执行文件为 `takeout-audit.exe`。无需安装 Go。
在解压后的程序目录打开 PowerShell，把下面的路径替换为你的实际目录：

```powershell
$parts = @(Get-ChildItem -LiteralPath 'D:\Takeout' -Filter '*.zip' -File | Sort-Object Name | ForEach-Object FullName)
.\takeout-audit.exe --inventory-only --out 'D:\Reports\inventory-01' @parts
```

请提供同一次导出的全部 ZIP。`D:\Reports` 要已存在，`inventory-01` 要是新目录。
离线模式的退出码 **2 是预期结果**，表示远端未核验；不是盘点程序崩溃。
也可以把 ZIP 参数换成已经解压的目录。不会把整组 ZIP 解压到硬盘。

## 连接 Immich

复制 `config.example.yaml` 为 `config.audit.yaml`，填写服务器地址和 API key。
地址不带 `/api`。密钥需要 `user.read` 和 `asset.upload` 权限，后者是查询接口的要求；
本命令通过网络请求白名单限制自身，只会查询。

```powershell
.\takeout-audit.exe --config config.audit.yaml --out 'D:\Reports\audit-01' @parts
```

首轮应使用隔离测试实例。不要将密钥贴到聊天或提交到代码仓库。

## 查看结果

先看 `summary.json`，再用表格软件筛选 `media.csv`；`ledger.jsonl` 是包含全部条目的原始账本。

| 结果 | 含义 |
|---|---|
| catalog_match | 找到相同内容校验值的记录，且明确不在回收站 |
| trashed | 匹配记录在回收站 |
| not_matched | 在当前账户的查询范围内没找到字节一致记录 |
| unknown / read_error | 无法判断或源文件读取失败，不能当作成功 |
| metadata_not_checked | 已记录 JSON 文件，未验证元数据迁移效果 |
| unclassified / excluded | 未识别格式或被明确排除的对象，原因保留 |

同一张照片出现在多个相册里会有多条源记录、一个内容组。摘要中的
`duplicate_media_copies` 表示额外副本数，`group_statuses` 按不同内容计数，
`statuses` 按源文件出现次数计数。重复或互相包含的输入路径会在读取前被拒绝。
“未匹配”不等于“丢失”：修改过元数据、重编码、另一个账户或外部库都可能影响结果。
“匹配”也不能证明服务器文件没有损坏，或拍摄时间、相册全部保留。

退出码：0＝本次范围内核对完成；1＝完成但有差异或需关注项；2＝未完成。
中断、读坏 ZIP、接口漏返回、权限不足都不会被显示为全部通过。
程序无法知道你是否漏选导出包。**不要仅凭本报告删除原件。**

默认上限为50万条目、单文件64GiB、总解压读取2TiB，可通过帮助中对应参数调整。
支持 Ctrl+C 后尽可能保存不完整报告；强制结束进程不保证落盘。

## 下一步验证

临时实例中的匹配、缺失、回收站三种状态已验证；真实用户导出数据和大规模性能仍待试用。
完整使用边界、测试入口和发布门槛见 [英文技术说明](docs/takeout-audit.md)。
独立程序没有上传、删除或自更新命令。开发包是否已公开、真实服务测试是否通过，
以仓库中的验证记录和实际 CI 运行结果为准；不能把测试配置完成当作联调通过。
