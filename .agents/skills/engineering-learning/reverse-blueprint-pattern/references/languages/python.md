# Python 参考

适合 FastAPI、Django、Flask、脚本、任务编排、轻量 agent。

## 默认顺序

1. `model` / schema / dataclass / Pydantic
2. service / use case / core function
3. route / command / worker / adapter
4. app / bootstrap / entrypoint

## 重点

- 先定数据模型，再定函数边界。
- service 层保持纯逻辑，不直接耦合框架对象。
- 路由层只负责解析、调用和响应。
- 同步和异步要提前选定，不要混用。
- 如果是 agent 或任务流，先写工具契约，再写编排。
