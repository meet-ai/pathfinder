# JobManagementContext 用例说明

## SubmitGoalUseCase — 主成功场景（带 daemon）

1. 用户在本机终端执行 `pathfinder -m "目标描述"` 提交一个新的高层目标。
2. CLI 将目标描述编码为 JSON 请求，通过 HTTP 调用本机常驻的 pathfinder daemon（`POST /jobs`），daemon 地址默认为 `http://127.0.0.1:8080`，可通过环境变量 `PATHFINDER_DAEMON_URL` 覆盖。
3. daemon 收到请求后加载配置、组装编排服务，调用 `WorkflowOrchestrationApplicationService.StartJob` 创建新的 Job 与对应的 Plan，并持久化初始状态。
4. daemon 在后台启动 `WorkflowOrchestrationApplicationService.ContinueJob`，按计划执行子任务、更新进度和 Job 状态。
5. daemon 为该 Job 生成状态查询 URL（例如 `http://127.0.0.1:8080/jobs/{jobId}`），并以 JSON 形式返回给 CLI，包含 `jobId` 与 `url` 字段。
6. CLI 从响应中解析出 URL，将该 URL 打印到终端，提示用户在浏览器中打开查看任务状态和执行情况。
7. 用户在浏览器中打开该 URL，以 JSON 形式查看当前 Job 的状态、任务列表和执行进度（后续可在前端或 TUI 上渲染为更友好的界面）。

## CancelJobUseCase — 主成功场景

1. 用户在浏览器中查看某个运行中的 job 状态页面，发现该任务不再需要继续执行，决定取消这个 job。
2. 用户在任务状态页面点击“取消”按钮，请求立即停止该 job 的后续执行。
3. 前端调用 daemon 的取消接口（例如 `POST /jobs/{jobId}/cancel`），daemon 将取消请求转交给工作流编排服务，标记该 job 为已请求取消。
4. 后续执行循环在下一次检查到取消标记时停止派发新的子任务，将 job 状态更新为已中止（aborted）。
5. 用户刷新任务状态页面后，看到该 job 的状态显示为“已取消”或“已中止”，不再有新的子任务执行。

## GetJobStateUseCase — 主成功场景（轮询获取状态）

1. 用户在前端界面或脚本中，希望定期获取某个 job 的当前状态与进度，用于刷新界面或做后续决策。
2. 前端或脚本根据手上的 `jobId`，以固定时间间隔向 daemon 发送 `GET /jobs/{jobId}` 请求。
3. daemon 收到请求后调用 `WorkflowOrchestrationApplicationService.GetJobState`，从 JobRepository 与 TaskProgressRepository 中恢复该 job 的当前状态、任务列表与进度。
4. daemon 将得到的 `JobStateDTO` 以 JSON 形式返回给前端或脚本。
5. 前端或脚本根据返回的数据更新界面上的进度条、任务列表与状态标签，或在脚本中根据 job 状态决定后续动作（继续轮询、停止轮询或触发取消）。 

