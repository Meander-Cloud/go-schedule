package scheduler

type Event interface {
	isEvent()
}

type exitEvent struct {
}

func (*exitEvent) isEvent() {}

type ReleaseGroupEvent[G comparable] struct {
	Group G
}

func (*ReleaseGroupEvent[G]) isEvent() {}

type ReleaseGroupSliceEvent[G comparable] struct {
	GroupSlice []G
}

func (*ReleaseGroupSliceEvent[G]) isEvent() {}

type ScheduleAsyncEvent[G comparable] struct {
	AsyncVariant *AsyncVariant[G]
}

func (*ScheduleAsyncEvent[G]) isEvent() {}

type ScheduleSequenceEvent[G comparable] struct {
	Sequence *Sequence[G]
}

func (*ScheduleSequenceEvent[G]) isEvent() {}
