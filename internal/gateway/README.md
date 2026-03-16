# gateway

HTTP/API、SSE/WS 流式；与 zeroclaw gateway 对齐。

## 职责

- HTTP/SSE/WS 入口
- Stream(RunId) 流式推送、Cancel(RunId) 取消
- 依赖 progress 读进度、orchestration 取消、config

## 对外提供 (provides)

- `StreamPublisher`、`StreamEvent`、`CancelRun`、`Server`
- provides.go 注释说明

## 外部依赖 (needs)

- TaskProgressRepository（读进度）
- orchestration.CancelRun
- config
- needs.go 说明

## 文件说明

| 文件 | 说明 |
|------|------|
| gateway.go | StreamPublisher、StreamEvent、CancelRun、Server |
| provides.go | 对外提供说明 |
| needs.go | 依赖说明 |
