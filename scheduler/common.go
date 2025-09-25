package scheduler

type LogProgressMode uint8

const (
	LogProgressModeNone LogProgressMode = 0
	LogProgressModeStep LogProgressMode = 1
	LogProgressModeRep  LogProgressMode = 2
)

type StepType uint8

const (
	StepTypeInvalid  StepType = 0
	StepTypeAction   StepType = 1
	StepTypeTimer    StepType = 2
	StepTypeSequence StepType = 3
)

func (t StepType) String() string {
	switch t {
	case StepTypeInvalid:
		return "invalid"
	case StepTypeAction:
		return "action"
	case StepTypeTimer:
		return "timer"
	case StepTypeSequence:
		return "sequence"
	default:
		return "unknown"
	}
}
