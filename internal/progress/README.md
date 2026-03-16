# progress

任务进度、checkpoint、恢复。

## 职责

- TaskProgress 实体、状态（pending/running/completed/failed/skipped）
- ProgressMaintainer：BatchUpdateProgress、Checkpoint、Restore
- 对外提供进度写入/读取与恢复

## 对外提供 (provides)

- `TaskStatus`、`Checkpoint`、`TaskProgress`（Start/Complete/Fail/WriteResult）
- `ProgressMaintainer`（BatchUpdateProgress、Checkpoint、Restore）
- provides.go 注释说明

## 外部依赖 (needs)

- `TaskProgressRepository`：Save、Get、ListByRunId，由 infra 实现
- needs.go 定义端口

## 文件说明

| 文件 | 说明 |
|------|------|
| task_progress.go | TaskStatus、Checkpoint、TaskProgress 及行为 |
| progress_maintainer.go | ProgressMaintainer 领域服务 |
| provides.go | 对外提供说明 |
| needs.go | TaskProgressRepository 端口 |
