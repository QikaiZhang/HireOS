# 中间件参考

适合协议网关、SDK、插件系统、消息中间件、缓存、中间层平台。

## 默认顺序

`protocol/spec -> codec/parser -> pipeline/plugin -> registry/config -> bootstrap`

## 重点

- 先定协议和兼容边界，再写实现。
- 关注性能、并发、背压、超时、重试和降级。
- 插件或扩展点要先定义生命周期。
- 配置、注册、发现和启动顺序必须明确。
- 兼容性和灰度策略要提前说明。
