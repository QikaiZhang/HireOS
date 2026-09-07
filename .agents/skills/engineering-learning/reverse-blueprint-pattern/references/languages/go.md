# Go 参考

适合 Go API、后台服务、worker、CLI、轻量 agent。

## 默认顺序

1. `struct` / entity / DTO / config
2. `service` interface + implementation
3. `handler` / router / middleware adapter
4. `main` / wiring / lifecycle

## 重点

- 先把 `json` / `db` / `validate` tag 说清楚。
- service 不碰 HTTP。
- handler 不写业务。
- 有存储时，把 repository / gateway 放在 service 之外。
- 错误要 wrap，别吞掉上下文。
- 用 table-driven tests 覆盖 service 和 handler。
