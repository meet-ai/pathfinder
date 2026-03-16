# runtime

执行环境（native/docker 等）；与 zeroclaw runtime 对齐。本包侧重 Run 生命周期与标识。

## 职责

- Run 聚合根：生命周期、取消、deadline
- 对外提供 Run 创建、取消、状态判断

## 对外提供 (provides)

- `RunId`、`StreamHandle`、`RunStatus`、`Run`
- `Create`、`Cancel`、`MarkAborted`、`IsCancelRequested`、`IsOverDeadline`
- provides.go 注释说明

## 外部依赖 (needs)

- `RunRepository`：Save/Get Run，由 infra 实现
- needs.go 定义端口

## 文件说明

| 文件 | 说明 |
|------|------|
| run.go | RunId、StreamHandle、RunStatus、Run 及行为 |
| provides.go | 对外提供说明 |
| needs.go | RunRepository 端口 |
