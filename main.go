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
				&scheduler.AsyncVariantArgs[uint8]{
					ReleaseGroup: true,
					GroupTracker: scheduler.NewGroupTracker([]uint8{1, 2}),
					Chan:         t1.C,
					SelectFunc: func(s *scheduler.Scheduler[uint8], v *scheduler.AsyncVariant[uint8], recv interface{}) {
						log.Printf("t1: selected, selectCount=%d, recv=%+v", v.SelectCount(), recv)
						s.ProcessSync(
							&scheduler.ReleaseGroupEvent[uint8]{
								Group: 1,
							},
						)
						log.Printf("t1: selected, done")
					},
					ReleaseFunc: func(s *scheduler.Scheduler[uint8], v *scheduler.AsyncVariant[uint8]) {
						log.Printf("t1: released, selectCount=%d", v.SelectCount())
						if v.SelectCount() > 0 {
							return
						}
						t1.Stop()
					},
				},
			),
		},
	)

	<-time.After(time.Second * 3)

	t2 := time.NewTimer(time.Second * 4)
	s.ProcessAsync(
		&scheduler.ScheduleAsyncEvent[uint8]{
			AsyncVariant: scheduler.NewAsyncVariant(
				&scheduler.AsyncVariantArgs[uint8]{
					ReleaseGroup: true,
					GroupTracker: scheduler.NewGroupTracker([]uint8{2, 3}),
					Chan:         t2.C,
					SelectFunc: func(s *scheduler.Scheduler[uint8], v *scheduler.AsyncVariant[uint8], recv interface{}) {
						log.Printf("t2: selected, selectCount=%d, recv=%+v", v.SelectCount(), recv)
					},
					ReleaseFunc: func(s *scheduler.Scheduler[uint8], v *scheduler.AsyncVariant[uint8]) {
						log.Printf("t2: released, selectCount=%d", v.SelectCount())
						if v.SelectCount() > 0 {
							return
						}
						t2.Stop()
					},
				},
			),
		},
	)

	<-time.After(time.Second * 3)

	t3 := time.NewTimer(time.Second)
	s.ProcessAsync(
		&scheduler.ScheduleAsyncEvent[string]{
			AsyncVariant: scheduler.NewAsyncVariant(
				&scheduler.AsyncVariantArgs[string]{
					ReleaseGroup: true,
					GroupTracker: scheduler.NewGroupTracker([]string{"A"}),
					Chan:         t3.C,
					SelectFunc:   nil,
					ReleaseFunc:  nil,
				},
			),
		},
	)

	<-time.After(time.Millisecond * 500)

	t4 := time.NewTimer(time.Second * 2)
	s.ProcessAsync(
		&scheduler.ScheduleAsyncEvent[uint8]{
			AsyncVariant: scheduler.NewAsyncVariant(
				&scheduler.AsyncVariantArgs[uint8]{
					ReleaseGroup: true,
					GroupTracker: scheduler.NewGroupTracker([]uint8{3, 4}),
					Chan:         t4.C,
					SelectFunc: func(s *scheduler.Scheduler[uint8], v *scheduler.AsyncVariant[uint8], recv interface{}) {
						log.Printf("t4: selected, selectCount=%d, recv=%+v", v.SelectCount(), recv)
					},
					ReleaseFunc: func(s *scheduler.Scheduler[uint8], v *scheduler.AsyncVariant[uint8]) {
						log.Printf("t4: released, selectCount=%d", v.SelectCount())
						if v.SelectCount() > 0 {
							return
						}
						t4.Stop()
					},
				},
			),
		},
	)

	t5 := time.NewTimer(time.Second * 3)
	s.ProcessAsync(
		&scheduler.ScheduleAsyncEvent[uint8]{
			AsyncVariant: scheduler.NewAsyncVariant(
				&scheduler.AsyncVariantArgs[uint8]{
					ReleaseGroup: false,
					GroupTracker: scheduler.NewGroupTracker([]uint8{4, 5}),
					Chan:         t5.C,
					SelectFunc: func(s *scheduler.Scheduler[uint8], v *scheduler.AsyncVariant[uint8], recv interface{}) {
						log.Printf("t5: selected, selectCount=%d, recv=%+v", v.SelectCount(), recv)
					},
					ReleaseFunc: func(s *scheduler.Scheduler[uint8], v *scheduler.AsyncVariant[uint8]) {
						log.Printf("t5: released, selectCount=%d", v.SelectCount())
						if v.SelectCount() > 0 {
							return
						}
						t5.Stop()
					},
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
				&scheduler.TimerAsyncArgs[uint8]{
					ReleaseGroup: false,
					GroupSlice:   []uint8{5, 6},
					Delay:        time.Second * 3,
					SelectFunc: func() {
						log.Printf("TimerAsync: selected")
					},
					ReleaseFunc: func(selectCount uint32) {
						log.Printf("TimerAsync: released, selectCount=%d", selectCount)
					},
				},
			),
		},
	)

	s.ProcessAsync(
		&scheduler.ScheduleAsyncEvent[uint8]{
			AsyncVariant: scheduler.TickerAsync(
				&scheduler.TickerAsyncArgs[uint8]{
					ReleaseGroup: false,
					GroupSlice:   []uint8{6, 7},
					Interval:     time.Second,
					SelectFunc: func() {
						log.Printf("TickerAsync: selected")
					},
					ReleaseFunc: func(selectCount uint32) {
						log.Printf("TickerAsync: released, selectCount=%d", selectCount)
					},
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
					&scheduler.SequenceArgs[string]{
						Scheduler:    s,
						InheritGroup: false,
						ReleaseGroup: true,
						GroupSlice:   []string{"A"},
						StepSlice: []*scheduler.Step[string]{
							scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: time.Second}),
							scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
								log.Printf("action 1")
								return nil
							}}),
							scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
								log.Printf("action 2")
								return nil
							}}),
							scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: time.Second * 2}),
							scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
								log.Printf("action 3")
								return nil
							}}),
						},
						StepResultFunc: func(p *scheduler.Step[string], result bool) {
							if result {
								log.Printf("group=%s, step<%d/%d>, type=%s, rep<%d/%d>, step completed", p.PrintGroup(), p.StepNum(), p.StepLen(), p.Descriptor(), p.RepNum(), p.RepTotal())
							} else {
								log.Printf("group=%s, step<%d/%d>, type=%s, rep<%d/%d>, step interrupted", p.PrintGroup(), p.StepNum(), p.StepLen(), p.Descriptor(), p.RepNum(), p.RepTotal())
							}
						},
						SequenceResultFunc: func(q *scheduler.Sequence[string], result bool) {
							if result {
								log.Printf("group=%s, step<%d/%d>, sequence completed", q.PrintGroup(), q.StepNum(), q.StepLen())
							} else {
								log.Printf("group=%s, step<%d/%d>, sequence interrupted", q.PrintGroup(), q.StepNum(), q.StepLen())
							}
						},
						LogProgressMode: scheduler.LogProgressModeRep,
					},
				),
			},
		)

		<-time.After(time.Second * 5)

		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					&scheduler.SequenceArgs[string]{
						Scheduler:    s,
						InheritGroup: false,
						ReleaseGroup: true,
						GroupSlice:   []string{"B", "C"},
						StepSlice: []*scheduler.Step[string]{
							scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: time.Second * 5}),
							scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
								log.Printf("action 1")
								return nil
							}}),
						},
						StepResultFunc: func(p *scheduler.Step[string], result bool) {
							if result {
								log.Printf("group=%s, step<%d/%d>, type=%s, rep<%d/%d>, step completed", p.PrintGroup(), p.StepNum(), p.StepLen(), p.Descriptor(), p.RepNum(), p.RepTotal())
							} else {
								log.Printf("group=%s, step<%d/%d>, type=%s, rep<%d/%d>, step interrupted", p.PrintGroup(), p.StepNum(), p.StepLen(), p.Descriptor(), p.RepNum(), p.RepTotal())
							}
						},
						SequenceResultFunc: func(q *scheduler.Sequence[string], result bool) {
							if result {
								log.Printf("group=%s, step<%d/%d>, sequence completed", q.PrintGroup(), q.StepNum(), q.StepLen())
							} else {
								log.Printf("group=%s, step<%d/%d>, sequence interrupted", q.PrintGroup(), q.StepNum(), q.StepLen())
							}
						},
						LogProgressMode: scheduler.LogProgressModeRep,
					},
				),
			},
		)

		<-time.After(time.Second * 2)

		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					&scheduler.SequenceArgs[string]{
						Scheduler:    s,
						InheritGroup: false,
						ReleaseGroup: true,
						GroupSlice:   []string{"C", "D"},
						StepSlice: []*scheduler.Step[string]{
							scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
								log.Printf("action 1")
								return nil
							}}),
							scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: time.Second * 2}),
							scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
								log.Printf("action 2")
								return nil
							}}),
							scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: time.Second * 3}),
						},
						StepResultFunc: func(p *scheduler.Step[string], result bool) {
							if result {
								log.Printf("group=%s, step<%d/%d>, type=%s, rep<%d/%d>, step completed", p.PrintGroup(), p.StepNum(), p.StepLen(), p.Descriptor(), p.RepNum(), p.RepTotal())
							} else {
								log.Printf("group=%s, step<%d/%d>, type=%s, rep<%d/%d>, step interrupted", p.PrintGroup(), p.StepNum(), p.StepLen(), p.Descriptor(), p.RepNum(), p.RepTotal())
							}
						},
						SequenceResultFunc: func(q *scheduler.Sequence[string], result bool) {
							if result {
								log.Printf("group=%s, step<%d/%d>, sequence completed", q.PrintGroup(), q.StepNum(), q.StepLen())
							} else {
								log.Printf("group=%s, step<%d/%d>, sequence interrupted", q.PrintGroup(), q.StepNum(), q.StepLen())
							}
						},
						LogProgressMode: scheduler.LogProgressModeRep,
					},
				),
			},
		)

		<-time.After(time.Second)

		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					&scheduler.SequenceArgs[string]{
						Scheduler:    s,
						InheritGroup: false,
						ReleaseGroup: false,
						GroupSlice:   []string{"D"},
						StepSlice: []*scheduler.Step[string]{
							scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
								log.Printf("action 1")
								return nil
							}}),
							scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: time.Second * 8}),
							scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
								log.Printf("action 2")
								return nil
							}}),
						},
						StepResultFunc: func(p *scheduler.Step[string], result bool) {
							if result {
								log.Printf("group=%s, step<%d/%d>, type=%s, rep<%d/%d>, step completed", p.PrintGroup(), p.StepNum(), p.StepLen(), p.Descriptor(), p.RepNum(), p.RepTotal())
							} else {
								log.Printf("group=%s, step<%d/%d>, type=%s, rep<%d/%d>, step interrupted", p.PrintGroup(), p.StepNum(), p.StepLen(), p.Descriptor(), p.RepNum(), p.RepTotal())
							}
						},
						SequenceResultFunc: func(q *scheduler.Sequence[string], result bool) {
							if result {
								log.Printf("group=%s, step<%d/%d>, sequence completed", q.PrintGroup(), q.StepNum(), q.StepLen())
							} else {
								log.Printf("group=%s, step<%d/%d>, sequence interrupted", q.PrintGroup(), q.StepNum(), q.StepLen())
							}
						},
						LogProgressMode: scheduler.LogProgressModeRep,
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
			LogPrefix: "test3",
			LogDebug:  true,
		},
	)

	step_rf := func(p *scheduler.Step[string], result bool) {
		log.Printf(
			"group=%s, step<%d/%d>, type=%s, rep<%d/%d>, step %s",
			p.PrintGroup(),
			p.StepNum(),
			p.StepLen(),
			p.Descriptor(),
			p.RepNum(),
			p.RepTotal(),
			func() string {
				if result {
					return "completed"
				} else {
					return "interrupted"
				}
			}(),
		)
	}

	seq_rf := func(q *scheduler.Sequence[string], result bool) {
		log.Printf(
			"group=%s, step<%d/%d>, sequence %s",
			q.PrintGroup(),
			q.StepNum(),
			q.StepLen(),
			func() string {
				if result {
					return "completed"
				} else {
					return "interrupted"
				}
			}(),
		)
	}

	go func() {
		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					&scheduler.SequenceArgs[string]{
						Scheduler:    s,
						InheritGroup: false,
						ReleaseGroup: true,
						GroupSlice:   []string{"A"},
						StepSlice: []*scheduler.Step[string]{
							scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: time.Second}),
							scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
								log.Printf("parent 1")
								return nil
							}}),
							scheduler.SequenceStep(
								&scheduler.SequenceStepArgs[string]{
									Sequence: scheduler.NewSequence(
										&scheduler.SequenceArgs[string]{
											Scheduler:    s,
											InheritGroup: false,
											ReleaseGroup: false,
											GroupSlice:   []string{"A", "B"},
											StepSlice: []*scheduler.Step[string]{
												scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
													log.Printf("child 1")
													return nil
												}}),
												scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: time.Millisecond * 500}),
											},
											StepResultFunc:     step_rf,
											SequenceResultFunc: seq_rf,
											LogProgressMode:    scheduler.LogProgressModeRep,
										},
									),
									RepTotal: 1,
								},
							),
							scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
								log.Printf("parent 2")
								return nil
							}}),
							scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: time.Second}),
							scheduler.SequenceStep(
								&scheduler.SequenceStepArgs[string]{
									Sequence: scheduler.NewSequence(
										&scheduler.SequenceArgs[string]{
											Scheduler:    s,
											InheritGroup: false,
											ReleaseGroup: true,
											GroupSlice:   []string{"A", "C"},
											StepSlice: []*scheduler.Step[string]{
												scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: time.Millisecond * 500}),
												scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
													log.Printf("child 2")
													return nil
												}}),
											},
											StepResultFunc:     step_rf,
											SequenceResultFunc: seq_rf,
											LogProgressMode:    scheduler.LogProgressModeRep,
										},
									),
									RepTotal: 1,
								},
							),
							scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
								log.Printf("parent 3")
								return nil
							}}),
						},
						StepResultFunc:     step_rf,
						SequenceResultFunc: seq_rf,
						LogProgressMode:    scheduler.LogProgressModeRep,
					},
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

	step_rf := func(p *scheduler.Step[string], result bool) {
		log.Printf(
			"group=%s, step<%d/%d>, type=%s, rep<%d/%d>, step %s",
			p.PrintGroup(),
			p.StepNum(),
			p.StepLen(),
			p.Descriptor(),
			p.RepNum(),
			p.RepTotal(),
			func() string {
				if result {
					return "completed"
				} else {
					return "interrupted"
				}
			}(),
		)
	}

	seq_rf := func(q *scheduler.Sequence[string], result bool) {
		log.Printf(
			"group=%s, step<%d/%d>, sequence %s",
			q.PrintGroup(),
			q.StepNum(),
			q.StepLen(),
			func() string {
				if result {
					return "completed"
				} else {
					return "interrupted"
				}
			}(),
		)
	}

	go func() {
		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					&scheduler.SequenceArgs[string]{
						Scheduler:    s,
						InheritGroup: false,
						ReleaseGroup: true,
						GroupSlice:   []string{"A"},
						StepSlice: []*scheduler.Step[string]{
							scheduler.SequenceStep(
								&scheduler.SequenceStepArgs[string]{
									Sequence: scheduler.NewSequence(
										&scheduler.SequenceArgs[string]{
											Scheduler:    s,
											InheritGroup: false,
											ReleaseGroup: false,
											GroupSlice:   []string{"B"},
											StepSlice: []*scheduler.Step[string]{
												scheduler.SequenceStep(
													&scheduler.SequenceStepArgs[string]{
														Sequence: scheduler.NewSequence(
															&scheduler.SequenceArgs[string]{
																Scheduler:    s,
																InheritGroup: false,
																ReleaseGroup: true,
																GroupSlice:   []string{"C"},
																StepSlice: []*scheduler.Step[string]{
																	scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: time.Second}),
																},
																StepResultFunc:     step_rf,
																SequenceResultFunc: seq_rf,
																LogProgressMode:    scheduler.LogProgressModeRep,
															},
														),
														RepTotal: 1,
													},
												),
											},
											StepResultFunc:     step_rf,
											SequenceResultFunc: seq_rf,
											LogProgressMode:    scheduler.LogProgressModeRep,
										},
									),
									RepTotal: 1,
								},
							),
						},
						StepResultFunc:     step_rf,
						SequenceResultFunc: seq_rf,
						LogProgressMode:    scheduler.LogProgressModeRep,
					},
				),
			},
		)

		<-time.After(time.Second * 3)

		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					&scheduler.SequenceArgs[string]{
						Scheduler:    s,
						InheritGroup: false,
						ReleaseGroup: true,
						GroupSlice:   []string{"A"},
						StepSlice: []*scheduler.Step[string]{
							scheduler.SequenceStep(
								&scheduler.SequenceStepArgs[string]{
									Sequence: scheduler.NewSequence(
										&scheduler.SequenceArgs[string]{
											Scheduler:          s,
											InheritGroup:       false,
											ReleaseGroup:       false,
											GroupSlice:         []string{"B"},
											StepSlice:          []*scheduler.Step[string]{},
											StepResultFunc:     step_rf,
											SequenceResultFunc: seq_rf,
											LogProgressMode:    scheduler.LogProgressModeRep,
										},
									),
									RepTotal: 1,
								},
							),
						},
						StepResultFunc:     step_rf,
						SequenceResultFunc: seq_rf,
						LogProgressMode:    scheduler.LogProgressModeRep,
					},
				),
			},
		)

		<-time.After(time.Second * 3)

		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					&scheduler.SequenceArgs[string]{
						Scheduler:          s,
						InheritGroup:       false,
						ReleaseGroup:       true,
						GroupSlice:         []string{"A"},
						StepSlice:          []*scheduler.Step[string]{},
						StepResultFunc:     step_rf,
						SequenceResultFunc: seq_rf,
						LogProgressMode:    scheduler.LogProgressModeRep,
					},
				),
			},
		)

		<-time.After(time.Second * 3)

		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					&scheduler.SequenceArgs[string]{
						Scheduler:    s,
						InheritGroup: false,
						ReleaseGroup: true,
						GroupSlice:   []string{"A"},
						StepSlice: []*scheduler.Step[string]{
							scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
								err := fmt.Errorf("parent action step fail")
								log.Printf("%s", err.Error())
								return err
							}}),
							scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
								log.Printf("cannot reach")
								return nil
							}}),
						},
						StepResultFunc:     step_rf,
						SequenceResultFunc: seq_rf,
						LogProgressMode:    scheduler.LogProgressModeRep,
					},
				),
			},
		)

		<-time.After(time.Second * 3)

		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					&scheduler.SequenceArgs[string]{
						Scheduler:    s,
						InheritGroup: false,
						ReleaseGroup: true,
						GroupSlice:   []string{"A"},
						StepSlice: []*scheduler.Step[string]{
							scheduler.SequenceStep(
								&scheduler.SequenceStepArgs[string]{
									Sequence: scheduler.NewSequence(
										&scheduler.SequenceArgs[string]{
											Scheduler:    s,
											InheritGroup: false,
											ReleaseGroup: false,
											GroupSlice:   []string{"B"},
											StepSlice: []*scheduler.Step[string]{
												scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: 0}),
												scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
													err := fmt.Errorf("child action step fail")
													log.Printf("%s", err.Error())
													return err
												}}),
												scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
													log.Printf("cannot reach")
													return nil
												}}),
											},
											StepResultFunc:     step_rf,
											SequenceResultFunc: seq_rf,
											LogProgressMode:    scheduler.LogProgressModeRep,
										},
									),
									RepTotal: 1,
								},
							),
							scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
								log.Printf("cannot reach")
								return nil
							}}),
						},
						StepResultFunc:     step_rf,
						SequenceResultFunc: seq_rf,
						LogProgressMode:    scheduler.LogProgressModeRep,
					},
				),
			},
		)

		<-time.After(time.Second * 3)

		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					&scheduler.SequenceArgs[string]{
						Scheduler:    s,
						InheritGroup: false,
						ReleaseGroup: true,
						GroupSlice:   []string{"A"},
						StepSlice: []*scheduler.Step[string]{
							scheduler.SequenceStep(
								&scheduler.SequenceStepArgs[string]{
									Sequence: scheduler.NewSequence(
										&scheduler.SequenceArgs[string]{
											Scheduler:    s,
											InheritGroup: false,
											ReleaseGroup: false,
											GroupSlice:   []string{"B"},
											StepSlice: []*scheduler.Step[string]{
												scheduler.SequenceStep(
													&scheduler.SequenceStepArgs[string]{
														Sequence: scheduler.NewSequence(
															&scheduler.SequenceArgs[string]{
																Scheduler:    s,
																InheritGroup: false,
																ReleaseGroup: true,
																GroupSlice:   []string{"C"},
																StepSlice: []*scheduler.Step[string]{
																	scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
																		log.Printf("child action step panic")

																		var err error
																		log.Printf("%s", err.Error())

																		return nil
																	}}),
																	scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
																		log.Printf("cannot reach")
																		return nil
																	}}),
																},
																StepResultFunc:     step_rf,
																SequenceResultFunc: seq_rf,
																LogProgressMode:    scheduler.LogProgressModeRep,
															},
														),
														RepTotal: 1,
													},
												),
											},
											StepResultFunc:     step_rf,
											SequenceResultFunc: seq_rf,
											LogProgressMode:    scheduler.LogProgressModeRep,
										},
									),
									RepTotal: 1,
								},
							),
						},
						StepResultFunc:     step_rf,
						SequenceResultFunc: seq_rf,
						LogProgressMode:    scheduler.LogProgressModeRep,
					},
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

	step_rf := func(p *scheduler.Step[string], result bool) {
		log.Printf(
			"group=%s, step<%d/%d>, type=%s, rep<%d/%d>, step %s",
			p.PrintGroup(),
			p.StepNum(),
			p.StepLen(),
			p.Descriptor(),
			p.RepNum(),
			p.RepTotal(),
			func() string {
				if result {
					return "completed"
				} else {
					return "interrupted"
				}
			}(),
		)
	}

	seq_rf := func(q *scheduler.Sequence[string], result bool) {
		log.Printf(
			"group=%s, step<%d/%d>, sequence %s",
			q.PrintGroup(),
			q.StepNum(),
			q.StepLen(),
			func() string {
				if result {
					return "completed"
				} else {
					return "interrupted"
				}
			}(),
		)
	}

	go func() {
		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					&scheduler.SequenceArgs[string]{
						Scheduler:    s,
						InheritGroup: false,
						ReleaseGroup: true,
						GroupSlice:   []string{"A1"},
						StepSlice: []*scheduler.Step[string]{
							scheduler.SequenceStep(
								&scheduler.SequenceStepArgs[string]{
									Sequence: scheduler.NewSequence(
										&scheduler.SequenceArgs[string]{
											Scheduler:    s,
											InheritGroup: false,
											ReleaseGroup: false,
											GroupSlice:   []string{"A2"},
											StepSlice: []*scheduler.Step[string]{
												scheduler.SequenceStep(
													&scheduler.SequenceStepArgs[string]{
														Sequence: scheduler.NewSequence(
															&scheduler.SequenceArgs[string]{
																Scheduler:    s,
																InheritGroup: false,
																ReleaseGroup: true,
																GroupSlice:   []string{"A3"},
																StepSlice: []*scheduler.Step[string]{
																	scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
																		log.Printf("action A3")
																		return nil
																	}}),
																},
																StepResultFunc:     step_rf,
																SequenceResultFunc: seq_rf,
																LogProgressMode:    scheduler.LogProgressModeRep,
															},
														),
														RepTotal: 2,
													},
												),
											},
											StepResultFunc:     step_rf,
											SequenceResultFunc: seq_rf,
											LogProgressMode:    scheduler.LogProgressModeRep,
										},
									),
									RepTotal: 2,
								},
							),
						},
						StepResultFunc:     step_rf,
						SequenceResultFunc: seq_rf,
						LogProgressMode:    scheduler.LogProgressModeRep,
					},
				),
			},
		)

		<-time.After(time.Second * 3)

		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					&scheduler.SequenceArgs[string]{
						Scheduler:    s,
						InheritGroup: false,
						ReleaseGroup: true,
						GroupSlice:   []string{"B1"},
						StepSlice: []*scheduler.Step[string]{
							scheduler.SequenceStep(
								&scheduler.SequenceStepArgs[string]{
									Sequence: scheduler.NewSequence(
										&scheduler.SequenceArgs[string]{
											Scheduler:    s,
											InheritGroup: false,
											ReleaseGroup: false,
											GroupSlice:   []string{"B2"},
											StepSlice: []*scheduler.Step[string]{
												scheduler.SequenceStep(
													&scheduler.SequenceStepArgs[string]{
														Sequence: scheduler.NewSequence(
															&scheduler.SequenceArgs[string]{
																Scheduler:    s,
																InheritGroup: false,
																ReleaseGroup: true,
																GroupSlice:   []string{"B3"},
																StepSlice: []*scheduler.Step[string]{
																	scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: time.Second}),
																},
																StepResultFunc:     step_rf,
																SequenceResultFunc: seq_rf,
																LogProgressMode:    scheduler.LogProgressModeRep,
															},
														),
														RepTotal: 2,
													},
												),
											},
											StepResultFunc:     step_rf,
											SequenceResultFunc: seq_rf,
											LogProgressMode:    scheduler.LogProgressModeRep,
										},
									),
									RepTotal: 2,
								},
							),
						},
						StepResultFunc:     step_rf,
						SequenceResultFunc: seq_rf,
						LogProgressMode:    scheduler.LogProgressModeRep,
					},
				),
			},
		)

		<-time.After(time.Second * 7)

		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					&scheduler.SequenceArgs[string]{
						Scheduler:    s,
						InheritGroup: false,
						ReleaseGroup: true,
						GroupSlice:   []string{"C1"},
						StepSlice: []*scheduler.Step[string]{
							scheduler.SequenceStep(
								&scheduler.SequenceStepArgs[string]{
									Sequence: scheduler.NewSequence(
										&scheduler.SequenceArgs[string]{
											Scheduler:    s,
											InheritGroup: false,
											ReleaseGroup: false,
											GroupSlice:   []string{"C2"},
											StepSlice: []*scheduler.Step[string]{
												scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
													log.Printf("action C2")
													return nil
												}}),
												scheduler.SequenceStep(
													&scheduler.SequenceStepArgs[string]{
														Sequence: scheduler.NewSequence(
															&scheduler.SequenceArgs[string]{
																Scheduler:    s,
																InheritGroup: false,
																ReleaseGroup: true,
																GroupSlice:   []string{"C3"},
																StepSlice: []*scheduler.Step[string]{
																	scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
																		log.Printf("action C3")
																		return nil
																	}}),
																	scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: time.Second}),
																},
																StepResultFunc:     step_rf,
																SequenceResultFunc: seq_rf,
																LogProgressMode:    scheduler.LogProgressModeRep,
															},
														),
														RepTotal: 2,
													},
												),
											},
											StepResultFunc:     step_rf,
											SequenceResultFunc: seq_rf,
											LogProgressMode:    scheduler.LogProgressModeRep,
										},
									),
									RepTotal: 2,
								},
							),
						},
						StepResultFunc:     step_rf,
						SequenceResultFunc: seq_rf,
						LogProgressMode:    scheduler.LogProgressModeRep,
					},
				),
			},
		)

		<-time.After(time.Second * 7)

		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					&scheduler.SequenceArgs[string]{
						Scheduler:    s,
						InheritGroup: false,
						ReleaseGroup: true,
						GroupSlice:   []string{"D1"},
						StepSlice: []*scheduler.Step[string]{
							scheduler.SequenceStep(
								&scheduler.SequenceStepArgs[string]{
									Sequence: scheduler.NewSequence(
										&scheduler.SequenceArgs[string]{
											Scheduler:    s,
											InheritGroup: false,
											ReleaseGroup: false,
											GroupSlice:   []string{"D2"},
											StepSlice: []*scheduler.Step[string]{
												scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
													log.Printf("action D2")
													return nil
												}}),
												scheduler.SequenceStep(
													&scheduler.SequenceStepArgs[string]{
														Sequence: scheduler.NewSequence(
															&scheduler.SequenceArgs[string]{
																Scheduler:    s,
																InheritGroup: false,
																ReleaseGroup: true,
																GroupSlice:   []string{"D3"},
																StepSlice: []*scheduler.Step[string]{
																	scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: time.Second}),
																	scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
																		log.Printf("action D3")
																		return nil
																	}}),
																},
																StepResultFunc:     step_rf,
																SequenceResultFunc: seq_rf,
																LogProgressMode:    scheduler.LogProgressModeRep,
															},
														),
														RepTotal: 2,
													},
												),
											},
											StepResultFunc:     step_rf,
											SequenceResultFunc: seq_rf,
											LogProgressMode:    scheduler.LogProgressModeRep,
										},
									),
									RepTotal: 2,
								},
							),
						},
						StepResultFunc:     step_rf,
						SequenceResultFunc: seq_rf,
						LogProgressMode:    scheduler.LogProgressModeRep,
					},
				),
			},
		)

		<-time.After(time.Second * 7)

		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					&scheduler.SequenceArgs[string]{
						Scheduler:    s,
						InheritGroup: false,
						ReleaseGroup: true,
						GroupSlice:   []string{"E1"},
						StepSlice: []*scheduler.Step[string]{
							scheduler.SequenceStep(
								&scheduler.SequenceStepArgs[string]{
									Sequence: scheduler.NewSequence(
										&scheduler.SequenceArgs[string]{
											Scheduler:    s,
											InheritGroup: false,
											ReleaseGroup: false,
											GroupSlice:   []string{"E2"},
											StepSlice: []*scheduler.Step[string]{
												scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: time.Second}),
												scheduler.SequenceStep(
													&scheduler.SequenceStepArgs[string]{
														Sequence: scheduler.NewSequence(
															&scheduler.SequenceArgs[string]{
																Scheduler:    s,
																InheritGroup: false,
																ReleaseGroup: true,
																GroupSlice:   []string{"E3"},
																StepSlice: []*scheduler.Step[string]{
																	scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
																		log.Printf("action E3")
																		return nil
																	}}),
																	scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: time.Second}),
																},
																StepResultFunc:     step_rf,
																SequenceResultFunc: seq_rf,
																LogProgressMode:    scheduler.LogProgressModeRep,
															},
														),
														RepTotal: 2,
													},
												),
											},
											StepResultFunc:     step_rf,
											SequenceResultFunc: seq_rf,
											LogProgressMode:    scheduler.LogProgressModeRep,
										},
									),
									RepTotal: 2,
								},
							),
						},
						StepResultFunc:     step_rf,
						SequenceResultFunc: seq_rf,
						LogProgressMode:    scheduler.LogProgressModeRep,
					},
				),
			},
		)

		<-time.After(time.Second * 9)

		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					&scheduler.SequenceArgs[string]{
						Scheduler:    s,
						InheritGroup: false,
						ReleaseGroup: true,
						GroupSlice:   []string{"F1"},
						StepSlice: []*scheduler.Step[string]{
							scheduler.SequenceStep(
								&scheduler.SequenceStepArgs[string]{
									Sequence: scheduler.NewSequence(
										&scheduler.SequenceArgs[string]{
											Scheduler:    s,
											InheritGroup: false,
											ReleaseGroup: false,
											GroupSlice:   []string{"F2"},
											StepSlice: []*scheduler.Step[string]{
												scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: time.Second}),
												scheduler.SequenceStep(
													&scheduler.SequenceStepArgs[string]{
														Sequence: scheduler.NewSequence(
															&scheduler.SequenceArgs[string]{
																Scheduler:    s,
																InheritGroup: false,
																ReleaseGroup: true,
																GroupSlice:   []string{"F3"},
																StepSlice: []*scheduler.Step[string]{
																	scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: time.Second}),
																	scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
																		log.Printf("action F3")
																		return nil
																	}}),
																},
																StepResultFunc:     step_rf,
																SequenceResultFunc: seq_rf,
																LogProgressMode:    scheduler.LogProgressModeRep,
															},
														),
														RepTotal: 2,
													},
												),
											},
											StepResultFunc:     step_rf,
											SequenceResultFunc: seq_rf,
											LogProgressMode:    scheduler.LogProgressModeRep,
										},
									),
									RepTotal: 2,
								},
							),
						},
						StepResultFunc:     step_rf,
						SequenceResultFunc: seq_rf,
						LogProgressMode:    scheduler.LogProgressModeRep,
					},
				),
			},
		)

		<-time.After(time.Second * 9)

		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					&scheduler.SequenceArgs[string]{
						Scheduler:    s,
						InheritGroup: false,
						ReleaseGroup: true,
						GroupSlice:   []string{"G1"},
						StepSlice: []*scheduler.Step[string]{
							scheduler.SequenceStep(
								&scheduler.SequenceStepArgs[string]{
									Sequence: scheduler.NewSequence(
										&scheduler.SequenceArgs[string]{
											Scheduler:    s,
											InheritGroup: false,
											ReleaseGroup: false,
											GroupSlice:   []string{"G2"},
											StepSlice: []*scheduler.Step[string]{
												scheduler.SequenceStep(
													&scheduler.SequenceStepArgs[string]{
														Sequence: scheduler.NewSequence(
															&scheduler.SequenceArgs[string]{
																Scheduler:    s,
																InheritGroup: false,
																ReleaseGroup: true,
																GroupSlice:   []string{"G3"},
																StepSlice: []*scheduler.Step[string]{
																	scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
																		log.Printf("action G3")
																		return nil
																	}}),
																	scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: time.Second}),
																},
																StepResultFunc:     step_rf,
																SequenceResultFunc: seq_rf,
																LogProgressMode:    scheduler.LogProgressModeRep,
															},
														),
														RepTotal: 2,
													},
												),
												scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
													log.Printf("action G2")
													return nil
												}}),
											},
											StepResultFunc:     step_rf,
											SequenceResultFunc: seq_rf,
											LogProgressMode:    scheduler.LogProgressModeRep,
										},
									),
									RepTotal: 2,
								},
							),
						},
						StepResultFunc:     step_rf,
						SequenceResultFunc: seq_rf,
						LogProgressMode:    scheduler.LogProgressModeRep,
					},
				),
			},
		)

		<-time.After(time.Second * 7)

		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					&scheduler.SequenceArgs[string]{
						Scheduler:    s,
						InheritGroup: false,
						ReleaseGroup: true,
						GroupSlice:   []string{"H1"},
						StepSlice: []*scheduler.Step[string]{
							scheduler.SequenceStep(
								&scheduler.SequenceStepArgs[string]{
									Sequence: scheduler.NewSequence(
										&scheduler.SequenceArgs[string]{
											Scheduler:    s,
											InheritGroup: false,
											ReleaseGroup: false,
											GroupSlice:   []string{"H2"},
											StepSlice: []*scheduler.Step[string]{
												scheduler.SequenceStep(
													&scheduler.SequenceStepArgs[string]{
														Sequence: scheduler.NewSequence(
															&scheduler.SequenceArgs[string]{
																Scheduler:    s,
																InheritGroup: false,
																ReleaseGroup: true,
																GroupSlice:   []string{"H3"},
																StepSlice: []*scheduler.Step[string]{
																	scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: time.Second}),
																	scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
																		log.Printf("action H3")
																		return nil
																	}}),
																},
																StepResultFunc:     step_rf,
																SequenceResultFunc: seq_rf,
																LogProgressMode:    scheduler.LogProgressModeRep,
															},
														),
														RepTotal: 2,
													},
												),
												scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
													log.Printf("action H2")
													return nil
												}}),
											},
											StepResultFunc:     step_rf,
											SequenceResultFunc: seq_rf,
											LogProgressMode:    scheduler.LogProgressModeRep,
										},
									),
									RepTotal: 2,
								},
							),
						},
						StepResultFunc:     step_rf,
						SequenceResultFunc: seq_rf,
						LogProgressMode:    scheduler.LogProgressModeRep,
					},
				),
			},
		)

		<-time.After(time.Second * 7)

		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					&scheduler.SequenceArgs[string]{
						Scheduler:    s,
						InheritGroup: false,
						ReleaseGroup: true,
						GroupSlice:   []string{"I1"},
						StepSlice: []*scheduler.Step[string]{
							scheduler.SequenceStep(
								&scheduler.SequenceStepArgs[string]{
									Sequence: scheduler.NewSequence(
										&scheduler.SequenceArgs[string]{
											Scheduler:    s,
											InheritGroup: false,
											ReleaseGroup: false,
											GroupSlice:   []string{"I2"},
											StepSlice: []*scheduler.Step[string]{
												scheduler.SequenceStep(
													&scheduler.SequenceStepArgs[string]{
														Sequence: scheduler.NewSequence(
															&scheduler.SequenceArgs[string]{
																Scheduler:    s,
																InheritGroup: false,
																ReleaseGroup: true,
																GroupSlice:   []string{"I3"},
																StepSlice: []*scheduler.Step[string]{
																	scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
																		log.Printf("action I3")
																		return nil
																	}}),
																	scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: time.Second}),
																},
																StepResultFunc:     step_rf,
																SequenceResultFunc: seq_rf,
																LogProgressMode:    scheduler.LogProgressModeRep,
															},
														),
														RepTotal: 2,
													},
												),
												scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: time.Second}),
											},
											StepResultFunc:     step_rf,
											SequenceResultFunc: seq_rf,
											LogProgressMode:    scheduler.LogProgressModeRep,
										},
									),
									RepTotal: 2,
								},
							),
						},
						StepResultFunc:     step_rf,
						SequenceResultFunc: seq_rf,
						LogProgressMode:    scheduler.LogProgressModeRep,
					},
				),
			},
		)

		<-time.After(time.Second * 9)

		s.ProcessAsync(
			&scheduler.ScheduleSequenceEvent[string]{
				Sequence: scheduler.NewSequence(
					&scheduler.SequenceArgs[string]{
						Scheduler:    s,
						InheritGroup: false,
						ReleaseGroup: true,
						GroupSlice:   []string{"J1"},
						StepSlice: []*scheduler.Step[string]{
							scheduler.SequenceStep(
								&scheduler.SequenceStepArgs[string]{
									Sequence: scheduler.NewSequence(
										&scheduler.SequenceArgs[string]{
											Scheduler:    s,
											InheritGroup: false,
											ReleaseGroup: false,
											GroupSlice:   []string{"J2"},
											StepSlice: []*scheduler.Step[string]{
												scheduler.SequenceStep(
													&scheduler.SequenceStepArgs[string]{
														Sequence: scheduler.NewSequence(
															&scheduler.SequenceArgs[string]{
																Scheduler:    s,
																InheritGroup: false,
																ReleaseGroup: true,
																GroupSlice:   []string{"J3"},
																StepSlice: []*scheduler.Step[string]{
																	scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: time.Second}),
																	scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
																		log.Printf("action J3")
																		return nil
																	}}),
																},
																StepResultFunc:     step_rf,
																SequenceResultFunc: seq_rf,
																LogProgressMode:    scheduler.LogProgressModeRep,
															},
														),
														RepTotal: 2,
													},
												),
												scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: time.Second}),
											},
											StepResultFunc:     step_rf,
											SequenceResultFunc: seq_rf,
											LogProgressMode:    scheduler.LogProgressModeRep,
										},
									),
									RepTotal: 2,
								},
							),
						},
						StepResultFunc:     step_rf,
						SequenceResultFunc: seq_rf,
						LogProgressMode:    scheduler.LogProgressModeRep,
					},
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

	step_rf := func(p *scheduler.Step[string], result bool) {
		log.Printf(
			"group=%s, step<%d/%d>, type=%s, rep<%d/%d>, step %s",
			p.PrintGroup(),
			p.StepNum(),
			p.StepLen(),
			p.Descriptor(),
			p.RepNum(),
			p.RepTotal(),
			func() string {
				if result {
					return "completed"
				} else {
					return "interrupted"
				}
			}(),
		)
	}

	seq_rf := func(q *scheduler.Sequence[string], result bool) {
		log.Printf(
			"group=%s, step<%d/%d>, sequence %s",
			q.PrintGroup(),
			q.StepNum(),
			q.StepLen(),
			func() string {
				if result {
					return "completed"
				} else {
					return "interrupted"
				}
			}(),
		)
	}

	go func() {
		q1 := scheduler.NewSequence(
			&scheduler.SequenceArgs[string]{
				Scheduler:    s,
				InheritGroup: false,
				ReleaseGroup: true,
				GroupSlice:   []string{"A1"},
				StepSlice: []*scheduler.Step[string]{
					scheduler.SequenceStep(
						&scheduler.SequenceStepArgs[string]{
							Sequence: scheduler.NewSequence(
								&scheduler.SequenceArgs[string]{
									Scheduler:    s,
									InheritGroup: false,
									ReleaseGroup: true,
									GroupSlice:   []string{"A2"},
									StepSlice: []*scheduler.Step[string]{
										scheduler.SequenceStep(
											&scheduler.SequenceStepArgs[string]{
												Sequence: scheduler.NewSequence(
													&scheduler.SequenceArgs[string]{
														Scheduler:    s,
														InheritGroup: false,
														ReleaseGroup: true,
														GroupSlice:   []string{"A3-1"},
														StepSlice: []*scheduler.Step[string]{
															scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
																log.Printf("action A3-1-1")
																return nil
															}}),
															scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
																log.Printf("action A3-1-2")
																return nil
															}}),
															scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
																log.Printf("action A3-1-3")
																return nil
															}}),
														},
														StepResultFunc:     step_rf,
														SequenceResultFunc: seq_rf,
														LogProgressMode:    scheduler.LogProgressModeRep,
													},
												),
												RepTotal: 2,
											},
										),
										scheduler.SequenceStep(
											&scheduler.SequenceStepArgs[string]{
												Sequence: scheduler.NewSequence(
													&scheduler.SequenceArgs[string]{
														Scheduler:    s,
														InheritGroup: false,
														ReleaseGroup: true,
														GroupSlice:   []string{"A3-2"},
														StepSlice: []*scheduler.Step[string]{
															scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: time.Millisecond * 200}),
															scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: time.Millisecond * 200}),
															scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: time.Millisecond * 200}),
														},
														StepResultFunc:     step_rf,
														SequenceResultFunc: seq_rf,
														LogProgressMode:    scheduler.LogProgressModeRep,
													},
												),
												RepTotal: 2,
											},
										),
									},
									StepResultFunc:     step_rf,
									SequenceResultFunc: seq_rf,
									LogProgressMode:    scheduler.LogProgressModeRep,
								},
							),
							RepTotal: 2,
						},
					),
				},
				StepResultFunc:     step_rf,
				SequenceResultFunc: seq_rf,
				LogProgressMode:    scheduler.LogProgressModeRep,
			},
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

func test7() {
	s := scheduler.NewScheduler[string](
		&scheduler.Options{
			LogPrefix: "test7",
			LogDebug:  true,
		},
	)

	s.Shutdown()
}

func test8() {
	s := scheduler.NewScheduler[string](
		&scheduler.Options{
			LogPrefix: "test8",
			LogDebug:  true,
		},
	)
	s.RunAsync()

	step_rf := func(p *scheduler.Step[string], result bool) {
		log.Printf(
			"group=%s, step<%d/%d>, type=%s, rep<%d/%d>, step %s",
			p.PrintGroup(),
			p.StepNum(),
			p.StepLen(),
			p.Descriptor(),
			p.RepNum(),
			p.RepTotal(),
			func() string {
				if result {
					return "completed"
				} else {
					return "interrupted"
				}
			}(),
		)
	}

	seq_rf := func(q *scheduler.Sequence[string], result bool) {
		log.Printf(
			"group=%s, step<%d/%d>, sequence %s",
			q.PrintGroup(),
			q.StepNum(),
			q.StepLen(),
			func() string {
				if result {
					return "completed"
				} else {
					return "interrupted"
				}
			}(),
		)
	}

	s.ProcessAsync(
		&scheduler.ScheduleSequenceEvent[string]{
			Sequence: scheduler.NewSequence(
				&scheduler.SequenceArgs[string]{
					Scheduler:    s,
					InheritGroup: false,
					ReleaseGroup: true,
					GroupSlice:   []string{"A1"},
					StepSlice: []*scheduler.Step[string]{
						scheduler.SequenceStep(
							&scheduler.SequenceStepArgs[string]{
								Sequence: scheduler.NewSequence(
									&scheduler.SequenceArgs[string]{
										Scheduler:    s,
										InheritGroup: false,
										ReleaseGroup: false,
										GroupSlice:   []string{"B1"},
										StepSlice: []*scheduler.Step[string]{
											scheduler.SequenceStep(
												&scheduler.SequenceStepArgs[string]{
													Sequence: scheduler.NewSequence(
														&scheduler.SequenceArgs[string]{
															Scheduler:    s,
															InheritGroup: false,
															ReleaseGroup: true,
															GroupSlice:   []string{"C1"},
															StepSlice: []*scheduler.Step[string]{
																scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: time.Second}),
															},
															StepResultFunc:     step_rf,
															SequenceResultFunc: seq_rf,
															LogProgressMode:    scheduler.LogProgressModeRep,
														},
													),
													RepTotal: 0,
												},
											),
										},
										StepResultFunc:     step_rf,
										SequenceResultFunc: seq_rf,
										LogProgressMode:    scheduler.LogProgressModeRep,
									},
								),
								RepTotal: 1,
							},
						),
					},
					StepResultFunc:     step_rf,
					SequenceResultFunc: seq_rf,
					LogProgressMode:    scheduler.LogProgressModeRep,
				},
			),
		},
	)

	<-time.After(time.Second * 2)

	s.ProcessAsync(
		&scheduler.ScheduleSequenceEvent[string]{
			Sequence: scheduler.NewSequence(
				&scheduler.SequenceArgs[string]{
					Scheduler:    s,
					InheritGroup: false,
					ReleaseGroup: true,
					GroupSlice:   []string{"A2"},
					StepSlice: []*scheduler.Step[string]{
						scheduler.SequenceStep(
							&scheduler.SequenceStepArgs[string]{
								Sequence: scheduler.NewSequence(
									&scheduler.SequenceArgs[string]{
										Scheduler:    s,
										InheritGroup: false,
										ReleaseGroup: false,
										GroupSlice:   []string{"B2"},
										StepSlice: []*scheduler.Step[string]{
											scheduler.SequenceStep(
												&scheduler.SequenceStepArgs[string]{
													Sequence: scheduler.NewSequence(
														&scheduler.SequenceArgs[string]{
															Scheduler:    s,
															InheritGroup: false,
															ReleaseGroup: true,
															GroupSlice:   []string{"C2"},
															StepSlice: []*scheduler.Step[string]{
																scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: time.Second}),
															},
															StepResultFunc:     step_rf,
															SequenceResultFunc: seq_rf,
															LogProgressMode:    scheduler.LogProgressModeRep,
														},
													),
													RepTotal: 1,
												},
											),
										},
										StepResultFunc:     step_rf,
										SequenceResultFunc: seq_rf,
										LogProgressMode:    scheduler.LogProgressModeRep,
									},
								),
								RepTotal: 0,
							},
						),
					},
					StepResultFunc:     step_rf,
					SequenceResultFunc: seq_rf,
					LogProgressMode:    scheduler.LogProgressModeRep,
				},
			),
		},
	)

	<-time.After(time.Second * 2)

	s.Shutdown()
}

func test9() {
	s := scheduler.NewScheduler[string](
		&scheduler.Options{
			LogPrefix: "test9",
			LogDebug:  true,
		},
	)
	s.RunAsync()

	step_rf := func(p *scheduler.Step[string], result bool) {
		log.Printf(
			"group=%s, step<%d/%d>, type=%s, rep<%d/%d>, step %s",
			p.PrintGroup(),
			p.StepNum(),
			p.StepLen(),
			p.Descriptor(),
			p.RepNum(),
			p.RepTotal(),
			func() string {
				if result {
					return "completed"
				} else {
					return "interrupted"
				}
			}(),
		)
	}

	seq_rf := func(q *scheduler.Sequence[string], result bool) {
		log.Printf(
			"group=%s, step<%d/%d>, sequence %s",
			q.PrintGroup(),
			q.StepNum(),
			q.StepLen(),
			func() string {
				if result {
					return "completed"
				} else {
					return "interrupted"
				}
			}(),
		)
	}

	s.ProcessAsync(
		&scheduler.ScheduleSequenceEvent[string]{
			Sequence: scheduler.NewSequence(
				&scheduler.SequenceArgs[string]{
					Scheduler:    s,
					InheritGroup: false,
					ReleaseGroup: true,
					GroupSlice:   []string{"B1"},
					StepSlice: []*scheduler.Step[string]{
						scheduler.SequenceStep(
							&scheduler.SequenceStepArgs[string]{
								Sequence: scheduler.NewSequence(
									&scheduler.SequenceArgs[string]{
										Scheduler:    s,
										InheritGroup: false,
										ReleaseGroup: true,
										GroupSlice:   []string{"B2"},
										StepSlice: []*scheduler.Step[string]{
											scheduler.SequenceStep(
												&scheduler.SequenceStepArgs[string]{
													Sequence: scheduler.NewSequence(
														&scheduler.SequenceArgs[string]{
															Scheduler:    s,
															InheritGroup: false,
															ReleaseGroup: true,
															GroupSlice:   []string{"B3-1"},
															StepSlice: []*scheduler.Step[string]{
																scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
																	log.Printf("action B3-1-1")
																	return nil
																}}),
																scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
																	log.Printf("action B3-1-2")
																	return nil
																}}),
																scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
																	log.Printf("action B3-1-3")
																	return nil
																}}),
															},
															StepResultFunc:     step_rf,
															SequenceResultFunc: seq_rf,
															LogProgressMode:    scheduler.LogProgressModeStep,
														},
													),
													RepTotal: 2,
												},
											),
											scheduler.SequenceStep(
												&scheduler.SequenceStepArgs[string]{
													Sequence: scheduler.NewSequence(
														&scheduler.SequenceArgs[string]{
															Scheduler:    s,
															InheritGroup: false,
															ReleaseGroup: true,
															GroupSlice:   []string{"B3-2"},
															StepSlice: []*scheduler.Step[string]{
																scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: time.Millisecond * 200}),
																scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: time.Millisecond * 200}),
																scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: time.Millisecond * 200}),
															},
															StepResultFunc:     step_rf,
															SequenceResultFunc: seq_rf,
															LogProgressMode:    scheduler.LogProgressModeStep,
														},
													),
													RepTotal: 2,
												},
											),
										},
										StepResultFunc:     step_rf,
										SequenceResultFunc: seq_rf,
										LogProgressMode:    scheduler.LogProgressModeStep,
									},
								),
								RepTotal: 2,
							},
						),
					},
					StepResultFunc:     step_rf,
					SequenceResultFunc: seq_rf,
					LogProgressMode:    scheduler.LogProgressModeStep,
				},
			),
		},
	)

	<-time.After(time.Second * 4)

	s.Shutdown()
}

func test10() {
	s := scheduler.NewScheduler[string](
		&scheduler.Options{
			LogPrefix: "test10",
			LogDebug:  false,
		},
	)
	s.RunAsync()

	seq_rf := func(q *scheduler.Sequence[string], result bool) {
		log.Printf(
			"group=%s, step<%d/%d>, sequence %s",
			q.PrintGroup(),
			q.StepNum(),
			q.StepLen(),
			func() string {
				if result {
					return "completed"
				} else {
					return "interrupted"
				}
			}(),
		)
	}

	s.ProcessAsync(
		&scheduler.ScheduleSequenceEvent[string]{
			Sequence: scheduler.NewSequence(
				&scheduler.SequenceArgs[string]{
					Scheduler:    s,
					InheritGroup: false,
					ReleaseGroup: true,
					GroupSlice:   []string{"C1"},
					StepSlice: []*scheduler.Step[string]{
						scheduler.SequenceStep(
							&scheduler.SequenceStepArgs[string]{
								Sequence: scheduler.NewSequence(
									&scheduler.SequenceArgs[string]{
										Scheduler:    s,
										InheritGroup: false,
										ReleaseGroup: true,
										GroupSlice:   []string{"C2"},
										StepSlice: []*scheduler.Step[string]{
											scheduler.SequenceStep(
												&scheduler.SequenceStepArgs[string]{
													Sequence: scheduler.NewSequence(
														&scheduler.SequenceArgs[string]{
															Scheduler:    s,
															InheritGroup: false,
															ReleaseGroup: true,
															GroupSlice:   []string{"C3-1"},
															StepSlice: []*scheduler.Step[string]{
																scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
																	log.Printf("action C3-1-1")
																	return nil
																}}),
																scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
																	log.Printf("action C3-1-2")
																	return nil
																}}),
																scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
																	log.Printf("action C3-1-3")
																	return nil
																}}),
															},
															StepResultFunc:     nil,
															SequenceResultFunc: seq_rf,
															LogProgressMode:    scheduler.LogProgressModeRep,
														},
													),
													RepTotal: 2,
												},
											),
											scheduler.SequenceStep(
												&scheduler.SequenceStepArgs[string]{
													Sequence: scheduler.NewSequence(
														&scheduler.SequenceArgs[string]{
															Scheduler:    s,
															InheritGroup: false,
															ReleaseGroup: true,
															GroupSlice:   []string{"C3-2"},
															StepSlice: []*scheduler.Step[string]{
																scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: time.Millisecond * 200}),
																scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: time.Millisecond * 200}),
																scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: time.Millisecond * 200}),
															},
															StepResultFunc:     nil,
															SequenceResultFunc: seq_rf,
															LogProgressMode:    scheduler.LogProgressModeRep,
														},
													),
													RepTotal: 2,
												},
											),
										},
										StepResultFunc:     nil,
										SequenceResultFunc: seq_rf,
										LogProgressMode:    scheduler.LogProgressModeRep,
									},
								),
								RepTotal: 2,
							},
						),
					},
					StepResultFunc:     nil,
					SequenceResultFunc: seq_rf,
					LogProgressMode:    scheduler.LogProgressModeRep,
				},
			),
		},
	)

	<-time.After(time.Second * 4)

	s.ProcessAsync(
		&scheduler.ScheduleSequenceEvent[string]{
			Sequence: scheduler.NewSequence(
				&scheduler.SequenceArgs[string]{
					Scheduler:    s,
					InheritGroup: false,
					ReleaseGroup: true,
					GroupSlice:   []string{"A"},
					StepSlice: []*scheduler.Step[string]{
						scheduler.SequenceStep(
							&scheduler.SequenceStepArgs[string]{
								Sequence: scheduler.NewSequence(
									&scheduler.SequenceArgs[string]{
										Scheduler:    s,
										InheritGroup: false,
										ReleaseGroup: false,
										GroupSlice:   []string{"B"},
										StepSlice: []*scheduler.Step[string]{
											scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
												err := fmt.Errorf("child action step fail")
												log.Printf("%s", err.Error())
												return err
											}}),
											scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
												log.Printf("cannot reach")
												return nil
											}}),
											scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: time.Second}),
										},
										StepResultFunc:     nil,
										SequenceResultFunc: seq_rf,
										LogProgressMode:    scheduler.LogProgressModeRep,
									},
								),
								RepTotal: 2,
							},
						),
						scheduler.ActionStep(&scheduler.ActionStepArgs[string]{Action: func() error {
							log.Printf("cannot reach")
							return nil
						}}),
					},
					StepResultFunc:     nil,
					SequenceResultFunc: seq_rf,
					LogProgressMode:    scheduler.LogProgressModeRep,
				},
			),
		},
	)

	<-time.After(time.Second * 2)

	s.Shutdown()
}

func test11() {
	s := scheduler.NewScheduler[string](
		&scheduler.Options{
			LogPrefix: "test11",
			LogDebug:  true,
		},
	)
	s.RunAsync()

	step_rf := func(p *scheduler.Step[string], result bool) {
		log.Printf(
			"group=%s, step<%d/%d>, type=%s, rep<%d/%d>, step %s",
			p.PrintGroup(),
			p.StepNum(),
			p.StepLen(),
			p.Descriptor(),
			p.RepNum(),
			p.RepTotal(),
			func() string {
				if result {
					return "completed"
				} else {
					return "interrupted"
				}
			}(),
		)
	}

	seq_rf := func(q *scheduler.Sequence[string], result bool) {
		log.Printf(
			"group=%s, step<%d/%d>, sequence %s",
			q.PrintGroup(),
			q.StepNum(),
			q.StepLen(),
			func() string {
				if result {
					return "completed"
				} else {
					return "interrupted"
				}
			}(),
		)
	}

	s.ProcessAsync(
		&scheduler.ScheduleSequenceEvent[string]{
			Sequence: scheduler.NewSequence(
				&scheduler.SequenceArgs[string]{
					Scheduler:    s,
					InheritGroup: true,
					ReleaseGroup: true,
					GroupSlice:   []string{"A1"},
					StepSlice: []*scheduler.Step[string]{
						scheduler.SequenceStep(
							&scheduler.SequenceStepArgs[string]{
								Sequence: scheduler.NewSequence(
									&scheduler.SequenceArgs[string]{
										Scheduler:    s,
										InheritGroup: true,
										ReleaseGroup: true,
										GroupSlice:   []string{"B1"},
										StepSlice: []*scheduler.Step[string]{
											scheduler.SequenceStep(
												&scheduler.SequenceStepArgs[string]{
													Sequence: scheduler.NewSequence(
														&scheduler.SequenceArgs[string]{
															Scheduler:    s,
															InheritGroup: true,
															ReleaseGroup: true,
															GroupSlice:   []string{"C1"},
															StepSlice: []*scheduler.Step[string]{
																scheduler.SequenceStep(
																	&scheduler.SequenceStepArgs[string]{
																		Sequence: scheduler.NewSequence(
																			&scheduler.SequenceArgs[string]{
																				Scheduler:    s,
																				InheritGroup: true,
																				ReleaseGroup: true,
																				GroupSlice:   []string{"D1"},
																				StepSlice: []*scheduler.Step[string]{
																					scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: time.Second}),
																				},
																				StepResultFunc:     step_rf,
																				SequenceResultFunc: seq_rf,
																				LogProgressMode:    scheduler.LogProgressModeRep,
																			},
																		),
																		RepTotal: 1,
																	},
																),
															},
															StepResultFunc:     step_rf,
															SequenceResultFunc: seq_rf,
															LogProgressMode:    scheduler.LogProgressModeRep,
														},
													),
													RepTotal: 1,
												},
											),
										},
										StepResultFunc:     step_rf,
										SequenceResultFunc: seq_rf,
										LogProgressMode:    scheduler.LogProgressModeRep,
									},
								),
								RepTotal: 2,
							},
						),
					},
					StepResultFunc:     step_rf,
					SequenceResultFunc: seq_rf,
					LogProgressMode:    scheduler.LogProgressModeRep,
				},
			),
		},
	)

	<-time.After(time.Second * 4)

	s.ProcessAsync(
		&scheduler.ScheduleSequenceEvent[string]{
			Sequence: scheduler.NewSequence(
				&scheduler.SequenceArgs[string]{
					Scheduler:    s,
					InheritGroup: true,
					ReleaseGroup: true,
					GroupSlice:   []string{"A2", "B2"},
					StepSlice: []*scheduler.Step[string]{
						scheduler.SequenceStep(
							&scheduler.SequenceStepArgs[string]{
								Sequence: scheduler.NewSequence(
									&scheduler.SequenceArgs[string]{
										Scheduler:    s,
										InheritGroup: true,
										ReleaseGroup: true,
										GroupSlice:   []string{"B2", "C2"},
										StepSlice: []*scheduler.Step[string]{
											scheduler.SequenceStep(
												&scheduler.SequenceStepArgs[string]{
													Sequence: scheduler.NewSequence(
														&scheduler.SequenceArgs[string]{
															Scheduler:    s,
															InheritGroup: true,
															ReleaseGroup: true,
															GroupSlice:   []string{},
															StepSlice: []*scheduler.Step[string]{
																scheduler.SequenceStep(
																	&scheduler.SequenceStepArgs[string]{
																		Sequence: scheduler.NewSequence(
																			&scheduler.SequenceArgs[string]{
																				Scheduler:    s,
																				InheritGroup: true,
																				ReleaseGroup: true,
																				GroupSlice:   nil,
																				StepSlice: []*scheduler.Step[string]{
																					scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: time.Second}),
																				},
																				StepResultFunc:     step_rf,
																				SequenceResultFunc: seq_rf,
																				LogProgressMode:    scheduler.LogProgressModeRep,
																			},
																		),
																		RepTotal: 1,
																	},
																),
															},
															StepResultFunc:     step_rf,
															SequenceResultFunc: seq_rf,
															LogProgressMode:    scheduler.LogProgressModeRep,
														},
													),
													RepTotal: 1,
												},
											),
										},
										StepResultFunc:     step_rf,
										SequenceResultFunc: seq_rf,
										LogProgressMode:    scheduler.LogProgressModeRep,
									},
								),
								RepTotal: 2,
							},
						),
					},
					StepResultFunc:     step_rf,
					SequenceResultFunc: seq_rf,
					LogProgressMode:    scheduler.LogProgressModeRep,
				},
			),
		},
	)

	<-time.After(time.Second * 4)

	s.ProcessAsync(
		&scheduler.ScheduleSequenceEvent[string]{
			Sequence: scheduler.NewSequence(
				&scheduler.SequenceArgs[string]{
					Scheduler:    s,
					InheritGroup: true,
					ReleaseGroup: true,
					GroupSlice:   nil,
					StepSlice: []*scheduler.Step[string]{
						scheduler.SequenceStep(
							&scheduler.SequenceStepArgs[string]{
								Sequence: scheduler.NewSequence(
									&scheduler.SequenceArgs[string]{
										Scheduler:    s,
										InheritGroup: true,
										ReleaseGroup: true,
										GroupSlice:   []string{"A3", "B3"},
										StepSlice: []*scheduler.Step[string]{
											scheduler.SequenceStep(
												&scheduler.SequenceStepArgs[string]{
													Sequence: scheduler.NewSequence(
														&scheduler.SequenceArgs[string]{
															Scheduler:    s,
															InheritGroup: false,
															ReleaseGroup: true,
															GroupSlice:   []string{"C3"},
															StepSlice: []*scheduler.Step[string]{
																scheduler.SequenceStep(
																	&scheduler.SequenceStepArgs[string]{
																		Sequence: scheduler.NewSequence(
																			&scheduler.SequenceArgs[string]{
																				Scheduler:    s,
																				InheritGroup: true,
																				ReleaseGroup: true,
																				GroupSlice:   []string{"D3"},
																				StepSlice: []*scheduler.Step[string]{
																					scheduler.TimerStep(&scheduler.TimerStepArgs[string]{Delay: time.Second}),
																				},
																				StepResultFunc:     step_rf,
																				SequenceResultFunc: seq_rf,
																				LogProgressMode:    scheduler.LogProgressModeRep,
																			},
																		),
																		RepTotal: 1,
																	},
																),
															},
															StepResultFunc:     step_rf,
															SequenceResultFunc: seq_rf,
															LogProgressMode:    scheduler.LogProgressModeRep,
														},
													),
													RepTotal: 1,
												},
											),
										},
										StepResultFunc:     step_rf,
										SequenceResultFunc: seq_rf,
										LogProgressMode:    scheduler.LogProgressModeRep,
									},
								),
								RepTotal: 2,
							},
						),
					},
					StepResultFunc:     step_rf,
					SequenceResultFunc: seq_rf,
					LogProgressMode:    scheduler.LogProgressModeRep,
				},
			),
		},
	)

	<-time.After(time.Second * 4)

	s.Shutdown()
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
	test7()
	test8()
	test9()
	test10()
	test11()
}
