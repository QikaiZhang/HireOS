# Output Structure

## `00-index.md`

包含项目一句话定位、当前阶段、文档导航、事实/假设/待确认列表、范围声明和变更日志。首页必须明确：这是增强方案，不是已完成生产系统的证明。

## `01-baseline.md`

用表格记录：模块、入口、主要调用方、数据存储、外部依赖、现有测试、部署方式、已观测问题、证据路径。没有证据的内容标为 `unknown`，不得补写成事实。

## `02-business-goal.md`

描述角色、核心业务对象、主流程、异常流程、人工介入点、权限边界、规模假设和不做项。把 AI 能力写成业务动作，例如“合同风险初筛后进入人工复核”，不要只写“接入 Agent”。

## `03-gap-analysis.md`

每条差距使用以下字段：`Gap`、`Observed evidence`、`Business impact`、`Priority`、`Proposed slice`、`Dependencies`、`Verification`。优先级建议使用 P0-P3，并区分现有问题、增强引入问题和被放大的问题。

## `04-target-architecture.md`

至少包含系统边界图、核心请求时序图、数据/事件流图和部署拓扑图。每张图说明读者、用途、关键假设、同步/异步边界和故障路径。

## `05-technology-decisions.md`

每个关键组件记录：问题、候选方案、选择、拒绝理由、版本来源、资源成本、运维复杂度、迁移/回滚策略。不要为了显得企业级强行引入 Kubernetes、向量库、消息队列或多 Agent。

## `06-risk-and-verification.md`

建立风险登记表、SLO/SLI、测试矩阵、故障演练清单和阶段验收门。验证项必须包含命令或操作、期望结果、实际结果和证据链接。

## `07-observability-and-operations.md`

说明日志字段、指标、Trace、告警、Dashboard、容量估算、发布、回滚、备份恢复和日常排障入口。AI 链路额外记录模型、提示版本、检索召回、Token、延迟和成本等信息。

## `08-interview-story.md`

只讲本人实际完成或明确验证的内容。每个亮点包含场景、约束、方案对比、实现、指标、失败复盘和后续改进；将“设计目标”和“实际结果”分开。
