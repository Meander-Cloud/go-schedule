package main

import (
	"fmt"
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
	s.ProcessAsync(
		&scheduler.ScheduleAsyncEvent[uint8]{
			AsyncVariant: scheduler.NewAsyncVariant(
				true,
				[]uint8{1, 2},
				t1.C,
				func(s *scheduler.Scheduler[uint8], v *scheduler.AsyncVariant[uint8], recv interface{}) {
					log.Printf("t1: selected, selectCount=%d, recv=%+v", v.SelectCount, recv)
					s.ProcessSync(
						&scheduler.ReleaseGroupEvent[uint8]{
							Group: 1,
						},
					)
					log.Printf("t1: selected, done")
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
	s.ProcessAsync(
		&scheduler.ScheduleAsyncEvent[uint8]{
			AsyncVariant: scheduler.NewAsyncVariant(
				true,
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
	s.ProcessAsync(
		&scheduler.ScheduleAsyncEvent[string]{
			AsyncVariant: scheduler.NewAsyncVariant(
				true,
				[]string{"A"},
				t3.C,
				nil,
				nil,
			),
		},
	)

	<-time.After(time.Millisecond * 500)

	t4 := time.NewTimer(time.Second * 2)
	s.ProcessAsync(
		&scheduler.ScheduleAsyncEvent[uint8]{
			AsyncVariant: scheduler.NewAsyncVariant(
				true,
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
	s.ProcessAsync(
		&scheduler.ScheduleAsyncEvent[uint8]{
			AsyncVariant: scheduler.NewAsyncVariant(
				false,
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

	s.ProcessAsync(
		&scheduler.ReleaseGroupSliceEvent[uint8]{
			GroupSlice: []uint8{5},
		},
	)

	<-time.After(time.Second * 3)

	s.ProcessAsync(
		&scheduler.ScheduleAsyncEvent[uint8]{
			AsyncVariant: scheduler.TimerAsync(
				false,
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

	s.ProcessAsync(
		&scheduler.ScheduleAsyncEvent[uint8]{
			AsyncVariant: scheduler.TickerAsync(
				false,
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
		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					s,
					true,
					[]string{"A"},
					[]*scheduler.Step[string]{
						scheduler.TimerStep[string](time.Second),
						scheduler.ActionStep[string](func() error {
							log.Printf("action 1")
							return nil
						}),
						scheduler.ActionStep[string](func() error {
							log.Printf("action 2")
							return nil
						}),
						scheduler.TimerStep[string](time.Second * 2),
						scheduler.ActionStep[string](func() error {
							log.Printf("action 3")
							return nil
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

		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					s,
					true,
					[]string{"B", "C"},
					[]*scheduler.Step[string]{
						scheduler.TimerStep[string](time.Second * 5),
						scheduler.ActionStep[string](func() error {
							log.Printf("action 1")
							return nil
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

		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					s,
					true,
					[]string{"C", "D"},
					[]*scheduler.Step[string]{
						scheduler.ActionStep[string](func() error {
							log.Printf("action 1")
							return nil
						}),
						scheduler.TimerStep[string](time.Second * 2),
						scheduler.ActionStep[string](func() error {
							log.Printf("action 2")
							return nil
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

		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					s,
					false,
					[]string{"D"},
					[]*scheduler.Step[string]{
						scheduler.ActionStep[string](func() error {
							log.Printf("action 1")
							return nil
						}),
						scheduler.TimerStep[string](time.Second * 8),
						scheduler.ActionStep[string](func() error {
							log.Printf("action 2")
							return nil
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

func test3() {
	s := scheduler.NewScheduler[string](
		&scheduler.Options{
			EventChannelLength: scheduler.EventChannelLength,
			LogPrefix:          "test3",
			LogDebug:           false,
		},
	)

	rf := func(s *scheduler.Sequence[string], stepResult bool, sequenceResult bool) {
		if !stepResult {
			log.Printf("group=%+v, sequence interrupted", s.GroupSlice)
			return
		}

		if sequenceResult {
			log.Printf("group=%+v, sequence completed", s.GroupSlice)
			return
		}
	}

	go func() {
		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					s,
					true,
					[]string{"A"},
					[]*scheduler.Step[string]{
						scheduler.TimerStep[string](time.Second),
						scheduler.ActionStep[string](func() error {
							log.Printf("parent 1")
							return nil
						}),
						scheduler.SequenceStep(
							scheduler.NewSequence(
								s,
								false,
								[]string{"A", "B"},
								[]*scheduler.Step[string]{
									scheduler.ActionStep[string](func() error {
										log.Printf("child 1")
										return nil
									}),
									scheduler.TimerStep[string](time.Millisecond * 500),
								},
								rf,
							),
						),
						scheduler.ActionStep[string](func() error {
							log.Printf("parent 2")
							return nil
						}),
						scheduler.TimerStep[string](time.Second),
						scheduler.SequenceStep(
							scheduler.NewSequence(
								s,
								true,
								[]string{"A", "C"},
								[]*scheduler.Step[string]{
									scheduler.TimerStep[string](time.Millisecond * 500),
									scheduler.ActionStep[string](func() error {
										log.Printf("child 2")
										return nil
									}),
								},
								rf,
							),
						),
						scheduler.ActionStep[string](func() error {
							log.Printf("parent 3")
							return nil
						}),
					},
					rf,
				),
			},
		)

		<-time.After(time.Second * 5)

		s.Shutdown()
	}()

	s.RunSync()
}

func test4() {
	s := scheduler.NewScheduler[string](
		&scheduler.Options{
			EventChannelLength: scheduler.EventChannelLength,
			LogPrefix:          "test4",
			LogDebug:           true,
		},
	)

	rf := func(s *scheduler.Sequence[string], stepResult bool, sequenceResult bool) {
		if !stepResult {
			log.Printf("group=%+v, sequence interrupted", s.GroupSlice)
			return
		}

		if sequenceResult {
			log.Printf("group=%+v, sequence completed", s.GroupSlice)
			return
		}
	}

	go func() {
		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					s,
					true,
					[]string{"A"},
					[]*scheduler.Step[string]{
						scheduler.SequenceStep(
							scheduler.NewSequence(
								s,
								false,
								[]string{"B"},
								[]*scheduler.Step[string]{
									scheduler.SequenceStep(
										scheduler.NewSequence(
											s,
											true,
											[]string{"C"},
											[]*scheduler.Step[string]{
												scheduler.TimerStep[string](time.Second),
											},
											rf,
										),
									),
								},
								rf,
							),
						),
					},
					rf,
				),
			},
		)

		<-time.After(time.Second * 3)

		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					s,
					true,
					[]string{"A"},
					[]*scheduler.Step[string]{
						scheduler.SequenceStep(
							scheduler.NewSequence(
								s,
								false,
								[]string{"B"},
								[]*scheduler.Step[string]{},
								rf,
							),
						),
					},
					rf,
				),
			},
		)

		<-time.After(time.Second * 3)

		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					s,
					true,
					[]string{"A"},
					[]*scheduler.Step[string]{},
					rf,
				),
			},
		)

		<-time.After(time.Second * 3)

		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					s,
					true,
					[]string{"A"},
					[]*scheduler.Step[string]{
						scheduler.ActionStep[string](func() error {
							err := fmt.Errorf("parent action step fail")
							log.Printf("%s", err.Error())
							return err
						}),
						scheduler.ActionStep[string](func() error {
							log.Printf("cannot reach")
							return nil
						}),
					},
					rf,
				),
			},
		)

		<-time.After(time.Second * 3)

		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					s,
					true,
					[]string{"A"},
					[]*scheduler.Step[string]{
						scheduler.SequenceStep(
							scheduler.NewSequence(
								s,
								false,
								[]string{"B"},
								[]*scheduler.Step[string]{
									//scheduler.TimerStep[string](0),
									scheduler.ActionStep[string](func() error {
										err := fmt.Errorf("child action step fail")
										log.Printf("%s", err.Error())
										return err
									}),
									scheduler.ActionStep[string](func() error {
										log.Printf("cannot reach")
										return nil
									}),
								},
								nil,
							),
						),
						scheduler.ActionStep[string](func() error {
							log.Printf("cannot reach")
							return nil
						}),
					},
					rf,
				),
			},
		)

		<-time.After(time.Second * 3)

		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					s,
					true,
					[]string{"A"},
					[]*scheduler.Step[string]{
						scheduler.SequenceStep(
							scheduler.NewSequence(
								s,
								false,
								[]string{"B"},
								[]*scheduler.Step[string]{
									scheduler.SequenceStep(
										scheduler.NewSequence(
											s,
											true,
											[]string{"C"},
											[]*scheduler.Step[string]{
												scheduler.ActionStep[string](func() error {
													log.Printf("child action step panic")

													var err error
													log.Printf("%s", err.Error())

													return nil
												}),
												scheduler.ActionStep[string](func() error {
													log.Printf("cannot reach")
													return nil
												}),
											},
											rf,
										),
									),
								},
								rf,
							),
						),
					},
					rf,
				),
			},
		)

		<-time.After(time.Second * 3)

		s.Shutdown()
	}()

	s.RunSync()
}

func main() {
	// enable microsecond and file line logging
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)

	test1()
	test2()
	test3()
	test4()
}
