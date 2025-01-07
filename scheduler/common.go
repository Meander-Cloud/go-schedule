package scheduler

const (
	EventChannelLength uint16 = 1024
)

type LogProgressMode uint8

const (
	LogProgressModeNone LogProgressMode = 0
	LogProgressModeStep LogProgressMode = 1
	LogProgressModeRep  LogProgressMode = 2
)
