# Java 参考

适合 Spring Boot、Web API、后台服务、消息消费、批处理。

## 默认顺序

1. `entity` / DTO / VO / config
2. `service` interface + `impl`
3. `controller` / request / response / advice
4. `bootstrap` / application / wiring

## 重点

- 优先明确注解：`@Entity`、`@Table`、`@Valid`、`@RequestBody` 等。
- 倾向构造器注入，不在业务里手写容器逻辑。
- controller 只做入参校验、转换、调用和响应封装。
- 异常映射和统一返回要单独收口。
- 持久化层与 service implementation 分开看。
