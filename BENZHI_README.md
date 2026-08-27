# BENZHI 评测说明

基于 Go 实现的历史广播节目单时段归属复核后端服务，一款后端服务，完成节目条目导入、时区钟差校正、时段归属与播出表冻结。

## 启动

```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go run ./cmd/broadcastslot --addr :8080 --db broadcastslot.db
```

## 自检（不启动长驻服务）

```bash
go run ./cmd/broadcastslot --smoke-test
```

`--smoke-test` 会真实创建批次与条目、执行钟差校正与对齐、记录裁决并冻结播出表版本，关闭并重新打开数据库验证持久化与 content_hash 不变，最后以 0 退出码结束。

## 构建门禁

```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go vet   ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go test  ./...
go run ./cmd/broadcastslot --smoke-test
```

## HTTP API（前缀 /api）

批次：`POST /api/batches`、`GET /api/batches`、`GET /api/batches/{id}`、`POST /api/batches/{id}/status`、`POST /api/batches/{id}/seal`
条目/片段/广告/引用：`POST/GET /api/batches/{id}/entries|clips|ads|citations`
校正/对齐：`POST /api/batches/{id}/correct`、`GET /api/batches/{id}/corrections`、`POST /api/batches/{id}/align`、`GET /api/batches/{id}/attributions`、`GET /api/batches/{id}/conflicts`
裁决/版本：`POST/GET /api/batches/{id}/verdicts`、`POST/GET /api/batches/{id}/versions`、`GET /api/batches/{id}/versions/{vid}`、`POST /api/batches/{id}/publish`
统计/健康：`GET /api/stats`、`GET /healthz`；复核页：`GET /`

## 持久化

SQLite（modernc.org/sqlite，CGO 无关）。表：evidence_batches、program_entries、station_clips、newspaper_ads、source_citations、clock_corrections、slot_attributions、attribution_conflicts、slot_verdicts、schedule_versions。条目指纹幂等；冻结版本 payload 不可变。
