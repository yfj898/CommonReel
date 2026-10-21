> Current execution and evidence: [IMPLEMENTATION_STATUS.md](IMPLEMENTATION_STATUS.md), [ACCEPTANCE.md](ACCEPTANCE.md). Original requirement/creation-time status below is retained as history.

# CommonReel Spec 需求与验收映射

日期：2026-10-13。依据 [正式需求](PRODUCT_REQUIREMENTS.md)、[实施 spec](SPEC.md) 与已确认 Q1–Q31。以下映射用于检查完整性，所有新 AT 的运行状态为 NOT_RUN。

| 需求主题 | 已确认决定 | User Stories | Spec 契约 | 验收 |
|---|---|---|---|---|
| 目标与比赛交付 | Q1–Q3、Q31 | 1–3、34 | Problem/Solution、阶段 G0–G4 | AT23–AT24 |
| 美国平台、片长与合并条件 | Q5、Q9、Q12、Q17 | 7–9、19–20 | Eligibility、US sub、未知失败、保存前刷新 | AT05–AT08、AT11 |
| 手机/局域网共享参与 | Q6–Q8 | 1–6 | LAN mode、房间凭据、SSE 即时快照/重连 | AT01–AT04、AT17–AT18 |
| 可代决定的记录语义 | Q11、Q16 | 4、28、31–32 | ActiveCandidate、AcceptanceRecord、匿名摘要 | AT09–AT10、AT16 |
| 空结果需明确调整 | Q13 | 23–24 | 过滤/未知原因、无静默放宽 | AT07–AT08 |
| Netflix CSV 与观看/喜欢区别 | Q14–Q15、Q21 | 10–14 | CSV 分类、严格匹配、manual resolve、明确偏好 | AT12–AT13 |
| 关键输入失效与并发保存 | Q18 | 5、28–30 | room_revision/decision_version、CAS、最小幂等回执 | AT03–AT04、AT09–AT11 |
| 真实本地代理与表单兜底 | Q19、Q23–Q24 | 25–27、33 | Ollama loop、真实 MCP、工具白名单/限额、页面最终动作 | AT19–AT21 |
| 免费真实数据及受限缓存 | Q20、Q29 | 19–22、33–34 | 目录 manifest、局部覆盖报告、配额/缓存、来源与实际状态 | AT06–AT08、AT22 |
| 跨场次画像及恢复码 | Q22、Q26–Q27 | 15–17 | 画像/场次副本、显式写回、恢复码权限 | AT02、AT12、AT14 |
| 画像撤回和场次生命周期 | Q28、Q30 | 18、32 | 删除事务、tombstone、缓存/回执清理、24 小时过期 | AT15–AT17、AT21 |
| 工程/真实数据/手机/群体证据 | Q25 | 34 | Testing Decisions、readiness、真实 vs adapter | AT20、AT22–AT24 |

Q4 的初始“全员本人接受”结果已由 Q11 修订；spec 不以代填记录证明个人意愿。Q10 的账号/完整历史输入选择已由 Q14 收敛为 CSV；Q22 的长期保存已由 Q26–Q28 具体化。Q23 明确零费用本地模型，原 20 元模型预算未接受。

## 现有能力与目标差距

| 可以复用的现有行为 | 本 spec 必须新增/调整 |
|---|---|
| MCP 2025-11-25 / Streamable HTTP 与真实 SDK integration test | 代理白名单、页面命令分工、真实本地模型循环 |
| SQLite WAL、事务 CAS、同请求幂等、重启恢复 | 双版本与接受失效、最小回执、权限/生命周期优先检查 |
| 类型否决与明确重看政策 | 时长/US 订阅供应、真实历史与来源、统一 eligibility |
| 估计 floor/average 聚合与目录先验 | 真人明确偏好输入、冷启动基线、popularity 标签修正；研究评价独立 |
| Go 静态页面和有限 SSE | 显式 LAN、手机兼容、header 授权、即时重连快照 |
| 原研究数据来源/留出与协议记录 | 200–300 目录标识、实际供应、CSV 解析/画像/删除、真实模型证据 |

这张表是代码复核后的实施差距，不表示右列功能已完成。需求 spec 变更时同时维护映射；运行证据由具体 AT 结果报告，不在此处补填未经执行的 PASS。


## 2026-10-21 扩展映射

| 已批准范围 | 实现/契约 | 验收 |
|---|---|---|
| 217 + 30 电影、可复现来源、旧 IMDb 已看 | 目录生成器、固定 seeds、规范身份比较 | AT22、AT27 |
| 可跳过的 1–3 明确喜欢、候选依据、原公式 | 现有 likes patch、preference_basis / estimate_basis | AT13、AT27 |
| 精确片长试算、有效缓存、零预览写入/供应请求 | runtime-preview HTTP / runtime_preview MCP | AT25 |
| 显式应用、当前版本/事实/到期绑定、幂等 | apply_runtime_preview + 原事务/回执 | AT03、AT10、AT25 |
| 宿主只读模式、普通入口不误写假设 | mode=runtime_preview + ordinary write guard | AT19、AT26 |
| 2/3/4 人、独立来源、手机尺寸/触摸/重连/双语/无模型表单 | Chromium 模拟；真机/真人状态单列 | AT17–AT20、AT23 |
| 同版本源码、英文视频、反馈、CI 与源码包 | 当前发布文档与 dist 交付清单 | AT24；上传及最终提交由用户接手 |
