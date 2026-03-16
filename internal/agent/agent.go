package agent

// AgentId 执行体唯一标识。
type AgentId string

// SkillId 技能唯一标识。
type SkillId string

// ToolId 工具唯一标识。
type ToolId string

// Agent 能力目录内执行体：谁可用、会什么（目录视角，不在此聚合内执行）。
type Agent struct {
	Id           AgentId
	Name         string
	Capabilities []string
	Tags         []string
}
