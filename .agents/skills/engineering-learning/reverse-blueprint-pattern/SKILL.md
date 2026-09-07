---
name: reverse-blueprint-pattern
description: Use this skill when you want AI-assisted implementation to follow a reverse-blueprint order, revealing code layer by layer instead of dumping a full solution at once. It keeps one stable entry point and adapts the same workflow across Go, Java, Python, backend, AI agent, and middleware tasks.
---

# 逆向施工图模式

用在你要让 AI 按物理依赖顺序逐层交付代码的时候。核心目标是先看骨架，再看装配，避免一次性黑盒输出。

## 选择规则

先确认任务类型，再按需要读参考文件。不要把语言和岗位拆成多个 skill 入口。

参考文件：

- [任务矩阵](references/profile-matrix.md)
- [通用语言差异](references/languages.md)
- [通用岗位差异](references/roles.md)
- [需求澄清问题集](references/intake.md)
- [逐步提示模板](references/prompts/blueprint_stepwise.md)

## 通用流程

1. 先让用户用自然语言描述项目目标、现状和约束，不要先发问卷。
2. 只在缺关键上下文时追问，单轮最多 3 个问题。
3. 需要澄清偏好时，优先问实现路线和代码风格，不要一次问太多维度。
4. 只输出当前层。
5. 每层结束后停下，等待用户确认或继续。
6. 不混写不同层的实现细节。
7. 先骨架，后依赖，最后装配。

## 默认层次

- 第 1 层：模型 / 实体 / DTO / 配置结构
- 第 2 层：Service / Use Case / Core
- 第 3 层：Handler / Controller / Router / Adapter
- 第 4 层：Bootstrap / wiring / main

## 纪律

- 不一次性输出全部代码。
- 不把 HTTP、SQL、消息队列、外部 API 混进核心业务层。
- 不跳过字段、错误码、依赖注入、启动装配的确认。
- 每层只回答该层的问题。
- 对用户偏好问题要短问短答，优先帮助用户明确取舍，而不是扩大讨论面。

## 提示语

可直接使用 [逐步提示模板](references/prompts/blueprint_stepwise.md) 里的模板。
