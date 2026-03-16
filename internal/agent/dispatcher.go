package agent

// Dispatcher 派发入口：编排层通过 needs.Dispatcher 端口派发子任务，由 infra 实现。
// 本包不实现派发逻辑，仅在 needs.go 中定义 Dispatcher 接口；具体实现见 infra/clients（如 DispatcherStub、dispatcher_openclaw）。
