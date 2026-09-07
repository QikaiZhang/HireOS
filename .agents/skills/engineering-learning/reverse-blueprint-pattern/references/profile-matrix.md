# 任务矩阵

先判断任务属于哪一类，再按需读取 `references/languages.md` 和 `references/roles.md`。

| 任务类型 | 关注点 | 默认交付顺序 |
| --- | --- | --- |
| 后端业务 | 数据流、事务、鉴权、存储、错误映射、观测性 | model -> service -> handler -> bootstrap |
| AI agent | 工具契约、规划、记忆、guardrails、evals | schema -> planner -> tools/adapter -> runtime |
| 中间件 | 协议、编码、管线、插件、兼容性 | spec -> codec -> pipeline -> bootstrap |
| 通用 Web/API | 结构、业务、接口、启动 | model -> service -> handler/controller -> main/app |

如果语言特性会影响层边界，再看 `references/languages.md`。
