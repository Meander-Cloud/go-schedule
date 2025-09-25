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
			LogPrefix: "test1",
			LogDebug:  true,
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
			LogPrefix: "test2",
			LogDebug:  true,
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
					func(q *scheduler.Sequence[string], stepIndex uint16, stepResult bool, sequenceResult bool) {
						if !stepResult {
							log.Printf("group=%+v, sequence interrupted at stepIndex=%d", q.GroupSlice, stepIndex)
							return
						}

						if sequenceResult {
							log.Printf("group=%+v, sequence completed", q.GroupSlice)
							return
						}
					},
					scheduler.LogProgressModeRep,
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
					func(q *scheduler.Sequence[string], stepIndex uint16, stepResult bool, sequenceResult bool) {
						if !stepResult {
							log.Printf("group=%+v, sequence interrupted at stepIndex=%d", q.GroupSlice, stepIndex)
							return
						}

						if sequenceResult {
							log.Printf("group=%+v, sequence completed", q.GroupSlice)
							return
						}
					},
					scheduler.LogProgressModeRep,
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
					func(q *scheduler.Sequence[string], stepIndex uint16, stepResult bool, sequenceResult bool) {
						if !stepResult {
							log.Printf("group=%+v, sequence interrupted at stepIndex=%d", q.GroupSlice, stepIndex)
							return
						}

						if sequenceResult {
							log.Printf("group=%+v, sequence completed", q.GroupSlice)
							return
						}
					},
					scheduler.LogProgressModeRep,
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
					func(q *scheduler.Sequence[string], stepIndex uint16, stepResult bool, sequenceResult bool) {
						if !stepResult {
							log.Printf("group=%+v, sequence interrupted at stepIndex=%d", q.GroupSlice, stepIndex)
							return
						}

						if sequenceResult {
							log.Printf("group=%+v, sequence completed", q.GroupSlice)
							return
						}
					},
					scheduler.LogProgressModeRep,
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
			LogPrefix: "test3",
			LogDebug:  true,
		},
	)

	rf := func(q *scheduler.Sequence[string], stepIndex uint16, stepResult bool, sequenceResult bool) {
		if !stepResult {
			log.Printf("group=%+v, sequence interrupted at stepIndex=%d", q.GroupSlice, stepIndex)
			return
		}

		if sequenceResult {
			log.Printf("group=%+v, sequence completed", q.GroupSlice)
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
							1,
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
								scheduler.LogProgressModeRep,
							),
						),
						scheduler.ActionStep[string](func() error {
							log.Printf("parent 2")
							return nil
						}),
						scheduler.TimerStep[string](time.Second),
						scheduler.SequenceStep(
							1,
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
								scheduler.LogProgressModeRep,
							),
						),
						scheduler.ActionStep[string](func() error {
							log.Printf("parent 3")
							return nil
						}),
					},
					rf,
					scheduler.LogProgressModeRep,
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
			LogPrefix: "test4",
			LogDebug:  true,
		},
	)

	rf := func(q *scheduler.Sequence[string], stepIndex uint16, stepResult bool, sequenceResult bool) {
		if !stepResult {
			log.Printf("group=%+v, sequence interrupted at stepIndex=%d", q.GroupSlice, stepIndex)
			return
		}

		if sequenceResult {
			log.Printf("group=%+v, sequence completed", q.GroupSlice)
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
							1,
							scheduler.NewSequence(
								s,
								false,
								[]string{"B"},
								[]*scheduler.Step[string]{
									scheduler.SequenceStep(
										1,
										scheduler.NewSequence(
											s,
											true,
											[]string{"C"},
											[]*scheduler.Step[string]{
												scheduler.TimerStep[string](time.Second),
											},
											rf,
											scheduler.LogProgressModeRep,
										),
									),
								},
								rf,
								scheduler.LogProgressModeRep,
							),
						),
					},
					rf,
					scheduler.LogProgressModeRep,
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
							1,
							scheduler.NewSequence(
								s,
								false,
								[]string{"B"},
								[]*scheduler.Step[string]{},
								rf,
								scheduler.LogProgressModeRep,
							),
						),
					},
					rf,
					scheduler.LogProgressModeRep,
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
					scheduler.LogProgressModeRep,
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
					scheduler.LogProgressModeRep,
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
							1,
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
								scheduler.LogProgressModeRep,
							),
						),
						scheduler.ActionStep[string](func() error {
							log.Printf("cannot reach")
							return nil
						}),
					},
					rf,
					scheduler.LogProgressModeRep,
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
							1,
							scheduler.NewSequence(
								s,
								false,
								[]string{"B"},
								[]*scheduler.Step[string]{
									scheduler.SequenceStep(
										1,
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
											scheduler.LogProgressModeRep,
										),
									),
								},
								rf,
								scheduler.LogProgressModeRep,
							),
						),
					},
					rf,
					scheduler.LogProgressModeRep,
				),
			},
		)

		<-time.After(time.Second * 3)

		s.Shutdown()
	}()

	s.RunSync()
}

func test5() {
	s := scheduler.NewScheduler[string](
		&scheduler.Options{
			LogPrefix: "test5",
			LogDebug:  true,
		},
	)

	rf := func(q *scheduler.Sequence[string], stepIndex uint16, stepResult bool, sequenceResult bool) {
		if !stepResult {
			log.Printf("group=%+v, sequence interrupted at stepIndex=%d", q.GroupSlice, stepIndex)
			return
		}

		if sequenceResult {
			log.Printf("group=%+v, sequence completed", q.GroupSlice)
			return
		}
	}

	go func() {
		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					s,
					true,
					[]string{"A1"},
					[]*scheduler.Step[string]{
						scheduler.SequenceStep(
							2,
							scheduler.NewSequence(
								s,
								false,
								[]string{"A2"},
								[]*scheduler.Step[string]{
									scheduler.SequenceStep(
										2,
										scheduler.NewSequence(
											s,
											true,
											[]string{"A3"},
											[]*scheduler.Step[string]{
												scheduler.ActionStep[string](func() error {
													log.Printf("action A3")
													return nil
												}),
											},
											rf,
											scheduler.LogProgressModeRep,
										),
									),
								},
								rf,
								scheduler.LogProgressModeRep,
							),
						),
					},
					rf,
					scheduler.LogProgressModeRep,
				),
			},
		)

		<-time.After(time.Second * 3)

		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					s,
					true,
					[]string{"B1"},
					[]*scheduler.Step[string]{
						scheduler.SequenceStep(
							2,
							scheduler.NewSequence(
								s,
								false,
								[]string{"B2"},
								[]*scheduler.Step[string]{
									scheduler.SequenceStep(
										2,
										scheduler.NewSequence(
											s,
											true,
											[]string{"B3"},
											[]*scheduler.Step[string]{
												scheduler.TimerStep[string](time.Second),
											},
											rf,
											scheduler.LogProgressModeRep,
										),
									),
								},
								rf,
								scheduler.LogProgressModeRep,
							),
						),
					},
					rf,
					scheduler.LogProgressModeRep,
				),
			},
		)

		<-time.After(time.Second * 7)

		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					s,
					true,
					[]string{"C1"},
					[]*scheduler.Step[string]{
						scheduler.SequenceStep(
							2,
							scheduler.NewSequence(
								s,
								false,
								[]string{"C2"},
								[]*scheduler.Step[string]{
									scheduler.ActionStep[string](func() error {
										log.Printf("action C2")
										return nil
									}),
									scheduler.SequenceStep(
										2,
										scheduler.NewSequence(
											s,
											true,
											[]string{"C3"},
											[]*scheduler.Step[string]{
												scheduler.ActionStep[string](func() error {
													log.Printf("action C3")
													return nil
												}),
												scheduler.TimerStep[string](time.Second),
											},
											rf,
											scheduler.LogProgressModeRep,
										),
									),
								},
								rf,
								scheduler.LogProgressModeRep,
							),
						),
					},
					rf,
					scheduler.LogProgressModeRep,
				),
			},
		)

		<-time.After(time.Second * 7)

		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					s,
					true,
					[]string{"D1"},
					[]*scheduler.Step[string]{
						scheduler.SequenceStep(
							2,
							scheduler.NewSequence(
								s,
								false,
								[]string{"D2"},
								[]*scheduler.Step[string]{
									scheduler.ActionStep[string](func() error {
										log.Printf("action D2")
										return nil
									}),
									scheduler.SequenceStep(
										2,
										scheduler.NewSequence(
											s,
											true,
											[]string{"D3"},
											[]*scheduler.Step[string]{
												scheduler.TimerStep[string](time.Second),
												scheduler.ActionStep[string](func() error {
													log.Printf("action D3")
													return nil
												}),
											},
											rf,
											scheduler.LogProgressModeRep,
										),
									),
								},
								rf,
								scheduler.LogProgressModeRep,
							),
						),
					},
					rf,
					scheduler.LogProgressModeRep,
				),
			},
		)

		<-time.After(time.Second * 7)

		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					s,
					true,
					[]string{"E1"},
					[]*scheduler.Step[string]{
						scheduler.SequenceStep(
							2,
							scheduler.NewSequence(
								s,
								false,
								[]string{"E2"},
								[]*scheduler.Step[string]{
									scheduler.TimerStep[string](time.Second),
									scheduler.SequenceStep(
										2,
										scheduler.NewSequence(
											s,
											true,
											[]string{"E3"},
											[]*scheduler.Step[string]{
												scheduler.ActionStep[string](func() error {
													log.Printf("action E3")
													return nil
												}),
												scheduler.TimerStep[string](time.Second),
											},
											rf,
											scheduler.LogProgressModeRep,
										),
									),
								},
								rf,
								scheduler.LogProgressModeRep,
							),
						),
					},
					rf,
					scheduler.LogProgressModeRep,
				),
			},
		)

		<-time.After(time.Second * 9)

		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					s,
					true,
					[]string{"F1"},
					[]*scheduler.Step[string]{
						scheduler.SequenceStep(
							2,
							scheduler.NewSequence(
								s,
								false,
								[]string{"F2"},
								[]*scheduler.Step[string]{
									scheduler.TimerStep[string](time.Second),
									scheduler.SequenceStep(
										2,
										scheduler.NewSequence(
											s,
											true,
											[]string{"F3"},
											[]*scheduler.Step[string]{
												scheduler.TimerStep[string](time.Second),
												scheduler.ActionStep[string](func() error {
													log.Printf("action F3")
													return nil
												}),
											},
											rf,
											scheduler.LogProgressModeRep,
										),
									),
								},
								rf,
								scheduler.LogProgressModeRep,
							),
						),
					},
					rf,
					scheduler.LogProgressModeRep,
				),
			},
		)

		<-time.After(time.Second * 9)

		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					s,
					true,
					[]string{"G1"},
					[]*scheduler.Step[string]{
						scheduler.SequenceStep(
							2,
							scheduler.NewSequence(
								s,
								false,
								[]string{"G2"},
								[]*scheduler.Step[string]{
									scheduler.SequenceStep(
										2,
										scheduler.NewSequence(
											s,
											true,
											[]string{"G3"},
											[]*scheduler.Step[string]{
												scheduler.ActionStep[string](func() error {
													log.Printf("action G3")
													return nil
												}),
												scheduler.TimerStep[string](time.Second),
											},
											rf,
											scheduler.LogProgressModeRep,
										),
									),
									scheduler.ActionStep[string](func() error {
										log.Printf("action G2")
										return nil
									}),
								},
								rf,
								scheduler.LogProgressModeRep,
							),
						),
					},
					rf,
					scheduler.LogProgressModeRep,
				),
			},
		)

		<-time.After(time.Second * 7)

		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					s,
					true,
					[]string{"H1"},
					[]*scheduler.Step[string]{
						scheduler.SequenceStep(
							2,
							scheduler.NewSequence(
								s,
								false,
								[]string{"H2"},
								[]*scheduler.Step[string]{
									scheduler.SequenceStep(
										2,
										scheduler.NewSequence(
											s,
											true,
											[]string{"H3"},
											[]*scheduler.Step[string]{
												scheduler.TimerStep[string](time.Second),
												scheduler.ActionStep[string](func() error {
													log.Printf("action H3")
													return nil
												}),
											},
											rf,
											scheduler.LogProgressModeRep,
										),
									),
									scheduler.ActionStep[string](func() error {
										log.Printf("action H2")
										return nil
									}),
								},
								rf,
								scheduler.LogProgressModeRep,
							),
						),
					},
					rf,
					scheduler.LogProgressModeRep,
				),
			},
		)

		<-time.After(time.Second * 7)

		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					s,
					true,
					[]string{"I1"},
					[]*scheduler.Step[string]{
						scheduler.SequenceStep(
							2,
							scheduler.NewSequence(
								s,
								false,
								[]string{"I2"},
								[]*scheduler.Step[string]{
									scheduler.SequenceStep(
										2,
										scheduler.NewSequence(
											s,
											true,
											[]string{"I3"},
											[]*scheduler.Step[string]{
												scheduler.ActionStep[string](func() error {
													log.Printf("action I3")
													return nil
												}),
												scheduler.TimerStep[string](time.Second),
											},
											rf,
											scheduler.LogProgressModeRep,
										),
									),
									scheduler.TimerStep[string](time.Second),
								},
								rf,
								scheduler.LogProgressModeRep,
							),
						),
					},
					rf,
					scheduler.LogProgressModeRep,
				),
			},
		)

		<-time.After(time.Second * 9)

		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					s,
					true,
					[]string{"J1"},
					[]*scheduler.Step[string]{
						scheduler.SequenceStep(
							2,
							scheduler.NewSequence(
								s,
								false,
								[]string{"J2"},
								[]*scheduler.Step[string]{
									scheduler.SequenceStep(
										2,
										scheduler.NewSequence(
											s,
											true,
											[]string{"J3"},
											[]*scheduler.Step[string]{
												scheduler.TimerStep[string](time.Second),
												scheduler.ActionStep[string](func() error {
													log.Printf("action J3")
													return nil
												}),
											},
											rf,
											scheduler.LogProgressModeRep,
										),
									),
									scheduler.TimerStep[string](time.Second),
								},
								rf,
								scheduler.LogProgressModeRep,
							),
						),
					},
					rf,
					scheduler.LogProgressModeRep,
				),
			},
		)

		<-time.After(time.Second * 9)

		s.Shutdown()
	}()

	s.RunSync()
}

func test6() {
	s := scheduler.NewScheduler[string](
		&scheduler.Options{
			LogPrefix: "test6",
			LogDebug:  true,
		},
	)

	rf := func(q *scheduler.Sequence[string], stepIndex uint16, stepResult bool, sequenceResult bool) {
		if !stepResult {
			log.Printf("group=%+v, sequence interrupted at stepIndex=%d", q.GroupSlice, stepIndex)
			return
		}

		if sequenceResult {
			log.Printf("group=%+v, sequence completed", q.GroupSlice)
			return
		}
	}

	go func() {
		q1 := scheduler.NewSequence(
			s,
			true,
			[]string{"A1"},
			[]*scheduler.Step[string]{
				scheduler.SequenceStep(
					2,
					scheduler.NewSequence(
						s,
						true,
						[]string{"A2"},
						[]*scheduler.Step[string]{
							scheduler.SequenceStep(
								2,
								scheduler.NewSequence(
									s,
									true,
									[]string{"A3-1"},
									[]*scheduler.Step[string]{
										scheduler.ActionStep[string](func() error {
											log.Printf("action A3-1-1")
											return nil
										}),
										scheduler.ActionStep[string](func() error {
											log.Printf("action A3-1-2")
											return nil
										}),
										scheduler.ActionStep[string](func() error {
											log.Printf("action A3-1-3")
											return nil
										}),
									},
									rf,
									scheduler.LogProgressModeRep,
								),
							),
							scheduler.SequenceStep(
								2,
								scheduler.NewSequence(
									s,
									true,
									[]string{"A3-2"},
									[]*scheduler.Step[string]{
										scheduler.TimerStep[string](time.Millisecond * 200),
										scheduler.TimerStep[string](time.Millisecond * 200),
										scheduler.TimerStep[string](time.Millisecond * 200),
									},
									rf,
									scheduler.LogProgressModeRep,
								),
							),
						},
						rf,
						scheduler.LogProgressModeRep,
					),
				),
			},
			rf,
			scheduler.LogProgressModeRep,
		)

		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: q1,
			},
		)

		<-time.After(time.Second * 4)

		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: q1,
			},
		)

		<-time.After(time.Second * 4)

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
	test5()
	test6()
}
