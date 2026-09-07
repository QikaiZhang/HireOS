# 通用语言差异

这里不拆成独立 skill，只记录语言层面最常见的结构差异。

## Go

- 常见层次：`struct` -> `service` -> `handler` -> `main`
- 重点：tags、error wrap、接口注入、table-driven tests

## Java

- 常见层次：`entity/DTO` -> `service` -> `controller` -> `bootstrap`
- 重点：注解、校验、构造器注入、统一异常映射

## Python

- 常见层次：`model/schema` -> `service/use case` -> `route` -> `app`
- 重点：Pydantic/dataclass、同步/异步边界、框架解耦
