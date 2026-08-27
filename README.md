# task278-broadcastslot

历史广播节目单时段归属复核后端：从残页条目、录音台呼与报纸广告证据出发，校正时区钟差、构造可行播出序列、识别广告延播冲突，并冻结不可变播出表版本。

## 目录结构

```
cmd/broadcastslot/     入口（HTTP 服务与 --smoke-test）
internal/
  model/               实体与领域错误
  evidence/            条目指纹（SHA-256 前 16 hex）
  clockfix/            时区与钟差校正
  sequence/            可行序列构造（Builder 可复用缓冲）
  conflict/            台呼重叠、广告延播、引用成环检测
  verdict/             归属裁决（乐观版本锁）
  snapshot/            播出表冻结深拷贝与 content_hash
  store/               SQLite 持久化
  service/             业务编排（serialMu 互斥对齐/发布/裁决）
  httpapi/             REST API 与复核页
  smoke/               端到端自检
```

## 本地运行

```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go run ./cmd/broadcastslot --addr :8080 --db broadcastslot.db
CGO_ENABLED=0 GOTOOLCHAIN=local go run ./cmd/broadcastslot --smoke-test
```

## 业务闭环

1. 建立证据批次（台名、播出日、时区、漂移 ppm）
2. 导入节目条目、台呼片段、报纸广告与来源互引
3. 钟差校正（`POST .../correct`）→ 序列对齐（`POST .../align`）
4. 裁决归属（`POST .../verdicts`）→ 构建并冻结播出表版本（`POST .../versions` + `POST .../publish`）

批次状态：organizing → pending_align → pending_verdict → published → sealed（封存只读）。

## 测试

```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go test ./...
```
