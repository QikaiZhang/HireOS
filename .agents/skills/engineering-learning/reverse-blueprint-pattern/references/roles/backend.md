# 后端参考

适合 CRUD、交易流、权限系统、订单、库存、审批、报表等业务。

## 默认顺序

`contract/model -> repository/gateway -> service/use case -> handler/controller -> bootstrap`

## 重点

- 先确定数据流和事务边界。
- 明确幂等性、错误码、鉴权、重试、超时。
- 业务层不要被 HTTP、SQL、消息队列污染。
- 观测性要单列出来：日志、指标、链路追踪。
- 有缓存或异步任务时，先说明一致性策略。
