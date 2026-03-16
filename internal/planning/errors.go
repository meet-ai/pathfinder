package planning

import "errors"

var (
	ErrPlanNoSubTasks       = errors.New("plan must have at least one subtask")
	ErrPlanInvalidDependency = errors.New("dependency must reference task ids within the same plan")
)
