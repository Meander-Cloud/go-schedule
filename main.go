package main

import (
	"log"
	"time"

	"github.com/Meander-Cloud/go-schedule/scheduler"
)

func test1() {
	s := scheduler.NewScheduler[uint8](
		&scheduler.Options{
			EventChannelLength: scheduler.EventChannelLength,
			LogPrefix:          "test1",
			LogDebug:           true,
		},
	)
	s.RunAsync()

	t1 := time.NewTimer(time.Second * 2)
	s.Process(
		&scheduler.ScheduleAsyncEvent[uint8]{
			ReleaseGroup: true,
			AsyncVariant: scheduler.NewAsyncVariant(
				[]uint8{1, 2},
				t1.C,
				func(s *scheduler.Scheduler[uint8], v *scheduler.AsyncVariant[uint8], recv interface{}) {
					log.Printf("t1: selected, selectCount=%d, recv=%+v", v.SelectCount, recv)
					s.Process(
						&scheduler.ReleaseGroupEvent[uint8]{
							Group: 1,
						},
					)
				},
				func(s *scheduler.Scheduler[uint8], v *scheduler.AsyncVariant[uint8]) {
					log.Printf("t1: released, selectCount=%d", v.SelectCount)
					if v.SelectCount > 0 {
						return
					}
					t1.Stop()
				},
			),
		},
	)

	<-time.After(time.Second * 3)

	t2 := time.NewTimer(time.Second * 4)
	s.Process(
		&scheduler.ScheduleAsyncEvent[uint8]{
			ReleaseGroup: true,
			AsyncVariant: scheduler.NewAsyncVariant(
				[]uint8{2, 3},
				t2.C,
				func(s *scheduler.Scheduler[uint8], v *scheduler.AsyncVariant[uint8], recv interface{}) {
					log.Printf("t2: selected, selectCount=%d, recv=%+v", v.SelectCount, recv)
				},
				func(s *scheduler.Scheduler[uint8], v *scheduler.AsyncVariant[uint8]) {
					log.Printf("t2: released, selectCount=%d", v.SelectCount)
					if v.SelectCount > 0 {
						return
					}
					t2.Stop()
				},
			),
		},
	)

	<-time.After(time.Second * 3)

	t3 := time.NewTimer(time.Second)
	s.Process(
		&scheduler.ScheduleAsyncEvent[string]{
			ReleaseGroup: true,
			AsyncVariant: scheduler.NewAsyncVariant(
				[]string{"A"},
				t3.C,
				nil,
				nil,
			),
		},
	)

	<-time.After(time.Millisecond * 500)

	t4 := time.NewTimer(time.Second * 2)
	s.Process(
		&scheduler.ScheduleAsyncEvent[uint8]{
			ReleaseGroup: true,
			AsyncVariant: scheduler.NewAsyncVariant(
				[]uint8{3, 4},
				t4.C,
				func(s *scheduler.Scheduler[uint8], v *scheduler.AsyncVariant[uint8], recv interface{}) {
					log.Printf("t4: selected, selectCount=%d, recv=%+v", v.SelectCount, recv)
				},
				func(s *scheduler.Scheduler[uint8], v *scheduler.AsyncVariant[uint8]) {
					log.Printf("t4: released, selectCount=%d", v.SelectCount)
					if v.SelectCount > 0 {
						return
					}
					t4.Stop()
				},
			),
		},
	)

	t5 := time.NewTimer(time.Second * 3)
	s.Process(
		&scheduler.ScheduleAsyncEvent[uint8]{
			ReleaseGroup: false,
			AsyncVariant: scheduler.NewAsyncVariant(
				[]uint8{4, 5},
				t5.C,
				func(s *scheduler.Scheduler[uint8], v *scheduler.AsyncVariant[uint8], recv interface{}) {
					log.Printf("t5: selected, selectCount=%d, recv=%+v", v.SelectCount, recv)
				},
				func(s *scheduler.Scheduler[uint8], v *scheduler.AsyncVariant[uint8]) {
					log.Printf("t5: released, selectCount=%d", v.SelectCount)
					if v.SelectCount > 0 {
						return
					}
					t5.Stop()
				},
			),
		},
	)

	<-time.After(time.Second * 5)

	s.Process(
		&scheduler.ReleaseGroupSliceEvent[uint8]{
			GroupSlice: []uint8{5},
		},
	)

	<-time.After(time.Second * 3)

	s.Process(
		&scheduler.ScheduleAsyncEvent[uint8]{
			ReleaseGroup: false,
			AsyncVariant: scheduler.TimerAsync(
				[]uint8{5, 6},
				time.Second*3,
				func() {
					log.Printf("TimerAsync: selected")
				},
				func(selectCount uint32) {
					log.Printf("TimerAsync: released, selectCount=%d", selectCount)
				},
			),
		},
	)

	s.Process(
		&scheduler.ScheduleAsyncEvent[uint8]{
			ReleaseGroup: false,
			AsyncVariant: scheduler.TickerAsync(
				[]uint8{6, 7},
				time.Second,
				func() {
					log.Printf("TickerAsync: selected")
				},
				func(selectCount uint32) {
					log.Printf("TickerAsync: released, selectCount=%d", selectCount)
				},
			),
		},
	)

	<-time.After(time.Second * 5)

	s.Shutdown()
}

func test2() {
	s := scheduler.NewScheduler[string](
		&scheduler.Options{
			EventChannelLength: scheduler.EventChannelLength,
			LogPrefix:          "test2",
			LogDebug:           false,
		},
	)

	go func() {
		s.Process(
			&scheduler.ScheduleSequenceEvent[string]{
				ReleaseGroup: true,
				Sequence: scheduler.NewSequence(
					s,
					[]string{"A"},
					[]*scheduler.Step[string]{
						scheduler.TimerStep[string](time.Second),
						scheduler.ActionStep[string](func() {
							log.Printf("action 1")
						}),
						scheduler.ActionStep[string](func() {
							log.Printf("action 2")
						}),
						scheduler.TimerStep[string](time.Second * 2),
						scheduler.ActionStep[string](func() {
							log.Printf("action 3")
						}),
					},
					func(s *scheduler.Sequence[string], stepResult bool, sequenceResult bool) {
						if !stepResult {
							log.Printf("group=%+v, sequence interrupted", s.GroupSlice)
							return
						}

						if sequenceResult {
							log.Printf("group=%+v, sequence completed", s.GroupSlice)
							return
						}
					},
				),
			},
		)

		<-time.After(time.Second * 5)

		s.Process(
			&scheduler.ScheduleSequenceEvent[string]{
				ReleaseGroup: true,
				Sequence: scheduler.NewSequence(
					s,
					[]string{"B", "C"},
					[]*scheduler.Step[string]{
						scheduler.TimerStep[string](time.Second * 5),
						scheduler.ActionStep[string](func() {
							log.Printf("action 1")
						}),
					},
					func(s *scheduler.Sequence[string], stepResult bool, sequenceResult bool) {
						if !stepResult {
							log.Printf("group=%+v, sequence interrupted", s.GroupSlice)
							return
						}

						if sequenceResult {
							log.Printf("group=%+v, sequence completed", s.GroupSlice)
							return
						}
					},
				),
			},
		)

		<-time.After(time.Second * 2)

		s.Process(
			&scheduler.ScheduleSequenceEvent[string]{
				ReleaseGroup: true,
				Sequence: scheduler.NewSequence(
					s,
					[]string{"C", "D"},
					[]*scheduler.Step[string]{
						scheduler.ActionStep[string](func() {
							log.Printf("action 1")
						}),
						scheduler.TimerStep[string](time.Second * 2),
						scheduler.ActionStep[string](func() {
							log.Printf("action 2")
						}),
						scheduler.TimerStep[string](time.Second * 3),
					},
					func(s *scheduler.Sequence[string], stepResult bool, sequenceResult bool) {
						if !stepResult {
							log.Printf("group=%+v, sequence interrupted", s.GroupSlice)
							return
						}

						if sequenceResult {
							log.Printf("group=%+v, sequence completed", s.GroupSlice)
							return
						}
					},
				),
			},
		)

		<-time.After(time.Second)

		s.Process(
			&scheduler.ScheduleSequenceEvent[string]{
				ReleaseGroup: false,
				Sequence: scheduler.NewSequence(
					s,
					[]string{"D"},
					[]*scheduler.Step[string]{
						scheduler.ActionStep[string](func() {
							log.Printf("action 1")
						}),
						scheduler.TimerStep[string](time.Second * 8),
						scheduler.ActionStep[string](func() {
							log.Printf("action 2")
						}),
					},
					func(s *scheduler.Sequence[string], stepResult bool, sequenceResult bool) {
						if !stepResult {
							log.Printf("group=%+v, sequence interrupted", s.GroupSlice)
							return
						}

						if sequenceResult {
							log.Printf("group=%+v, sequence completed", s.GroupSlice)
							return
						}
					},
				),
			},
		)

		<-time.After(time.Second * 7)

		s.Shutdown()
	}()

	s.RunSync()
}

func main() {
	// enable microsecond and file line logging
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)

	test1()
	test2()
}
