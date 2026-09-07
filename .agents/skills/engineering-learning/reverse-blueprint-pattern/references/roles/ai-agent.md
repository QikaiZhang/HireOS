# AI agent 参考

适合需要工具调用、规划、记忆、反思、评测和安全约束的 agent 工程。

## 默认顺序

`goal/router -> tool schema -> planner/executor -> memory/state -> guardrails/evals -> runtime`

## 重点

- 先定义工具契约，再定义调用流程。
- 明确 planner 和 worker 的职责边界。
- 记忆、状态、缓存、追踪要分清。
- 处理 prompt injection、越权调用、预算和失败恢复。
- 输出结构要稳定，便于自动评测和回放。
