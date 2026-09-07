# Engineering Learning Skills

这一组 Skill 面向真实企业项目中的长期工程成长：既要快速读懂和修改代码，也要把值得保留的设计、取舍和项目经验沉淀下来。它们不绑定特定框架，适用于 Go、Java、Python、TypeScript 等后端项目。

## Skill 列表

| Skill | 适用时机 | 主要产出 |
| --- | --- | --- |
| [code-analysis-coach](./code-analysis-coach/SKILL.md) | 接手陌生模块、追踪调用链、定位改动、读内部基础库、提炼面试素材 | 模块地图、真实调用链、修改风险、按价值生成的学习与面试资料 |
| [reverse-blueprint-pattern](./reverse-blueprint-pattern/SKILL.md) | 从零实现或重构后端能力，需要控制依赖顺序和交付边界 | 从基础设施到业务功能的分层实现路径 |
| [incremental-branch-training](./incremental-branch-training/SKILL.md) | 需要通过连续需求练习提升实现、审查和项目讲解能力 | 增量任务、审查反馈、复盘和项目表达训练 |

## 如何选择

- **代码已经存在，需要理解或修改：** 从 `code-analysis-coach` 开始。它优先建立真实调用链，避免逐行翻译和盲目阅读。
- **能力尚未实现，需要从基础开始设计与交付：** 使用 `reverse-blueprint-pattern`，按物理依赖逐层完成。
- **希望把做项目变成可验证的训练：** 使用 `incremental-branch-training`，把需求拆分、实现、审查和复盘串起来。

常见闭环是：`读现有代码 → 设计/实现改动 → 审查与复盘 → 形成可讲述的工程经验`。三个 Skill 的侧重点不同，避免重复启用同类流程。
