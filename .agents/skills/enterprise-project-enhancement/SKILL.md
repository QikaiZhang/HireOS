---
name: enterprise-project-enhancement
description: 根据现有项目和目标，生成一套可执行、可验证、面向真实企业场景与秋招面试的项目增强文档仓库；适用于把 Demo 渐进增强为有业务闭环、工程证据和技术取舍的中大型项目方案，不负责直接实现项目代码。
metadata:
  version: "1.0.0"
  short-description: "从现有项目生成企业级增强方案与阶段文档仓库"
---

# Enterprise Project Enhancement

## 核心定位

把现有 Demo 或普通项目升级为“接近真实企业场景、但不脱离现有基础”的增强路线图和证据仓库。输出重点是：

```text
现状证据 -> 目标差距 -> 增强切片 -> 阶段交付 -> 验证证据 -> 面试叙事
```

本 Skill 只产出文档和规划，不直接修改业务代码、部署环境或依赖配置。除非用户另行授权，不执行项目实现、迁移、压测或线上操作。

## 使用边界

- 先阅读现有项目，再提出增强方案；不得凭技术栈清单臆造项目现状。
- “企业级”必须由业务约束、失败场景、数据规模、SLO、权限边界、可观测性和验证证据共同定义，不能靠罗列组件定义。
- 保留原项目的主业务和技术资产。每项增强都必须说明与现有代码的连接点、增量范围、价值和验证方式。
- AI、微服务、分布式、集群等技术只有在解决明确业务问题时引入；不为了简历堆技术，不把不可验证的指标写成已达成结果。
- 目标是接近 T1/T0 企业工程思路，不承诺把小型 Demo 伪装成生产系统；对无法在当前项目规模验证的能力标记为 `planned` 或 `simulation`。

## 工作流程

1. **建立现状基线**：检查目录、入口、调用链、数据模型、依赖、配置、测试、部署文件、日志和已有文档。记录事实、证据路径和不确定项。
2. **定义业务目标**：从用户、业务对象、核心流程、异常流程和规模约束出发，选择一个聚焦的企业子场景；明确不做什么。
3. **做差距与分级**：把差距分为业务闭环、架构边界、可靠性、性能、数据一致性、安全、可观测性、交付与面试证据；按影响和可验证性排序。
4. **设计增量蓝图**：给出目标架构、数据流、部署拓扑和关键时序；每个组件必须有选择理由、替代方案、成本和回滚方式。版本信息需核对对应官方文档，不直接把示例版本当作事实。
5. **拆分阶段交付**：每个阶段包含目标、前置条件、修改范围、产物、验证命令/指标、风险、停止条件和面试素材。阶段应能独立形成可演示闭环。
6. **建立证据链**：为每个重要结论绑定代码路径、配置、测试、压测结果、截图或实验记录；区分 `observed`、`implemented`、`verified`、`planned`。
7. **输出复盘与宣讲包**：生成技术决策记录、问题复盘、量化指标表、STAR-L 故事线和分层追问，不夸大个人贡献或实验结果。

## 默认文档仓库

在目标项目中生成独立目录（默认 `docs/project-enhancement/`，若项目已有文档约定则遵循项目约定）：

```text
docs/project-enhancement/
├── 00-index.md                 # 导航、状态、阅读顺序、事实/假设声明
├── 01-baseline.md              # 现状基线与证据索引
├── 02-business-goal.md         # 场景、角色、业务闭环、范围边界
├── 03-gap-analysis.md          # 差距、优先级、约束和不做项
├── 04-target-architecture.md   # 架构、数据流、部署拓扑、时序图
├── 05-technology-decisions.md  # 技术选型、替代方案、版本与成本
├── phases/
│   ├── phase-0-baseline.md
│   ├── phase-1-business-closed-loop.md
│   ├── phase-2-reliability.md
│   ├── phase-3-performance.md
│   ├── phase-4-ai-or-intelligence.md
│   └── phase-5-production-simulation.md
├── 06-risk-and-verification.md # 风险、SLO、测试矩阵、验收门
├── 07-observability-and-operations.md
├── 08-interview-story.md        # STAR-L、贡献边界、追问准备
└── evidence/
    ├── evidence-index.md
    └── templates.md
```

详细字段和阶段模板见：

- [references/output-structure.md](references/output-structure.md)
- [references/phase-documents.md](references/phase-documents.md)
- [references/architecture-and-evidence.md](references/architecture-and-evidence.md)
- [references/interview-package.md](references/interview-package.md)

## 质量门

交付前逐项检查：

- 每个目标能力都能追溯到现状问题和业务价值。
- 每个阶段都有明确产物、验证方式和停止条件。
- 架构图与文字、代码入口、配置和数据流一致。
- 指标区分基线、目标、实测和模拟；没有把预估数据写成结果。
- 技术选型包含至少一个替代方案和取舍理由；版本以官方资料或项目锁定版本为准。
- 失败、重试、幂等、超时、降级、权限和审计路径有说明。
- 文档能支持维护者实施，也能支持面试者解释“为什么这样设计”。

## 输出风格

优先使用 Markdown 表格、Mermaid `flowchart`/`sequenceDiagram`/`erDiagram` 和短段落。图必须服务于决策或关系表达，并在图下补充关键假设；避免装饰性大图。所有结论注明来源：代码路径、配置路径、实验记录、官方文档或明确假设。
