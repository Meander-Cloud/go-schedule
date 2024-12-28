package scheduler

import (
	"log"
	"time"
)

type Step[G comparable] struct {
	resolve func(*Sequence[G]) (bool, func())
}

type Sequence[G comparable] struct {
	scheduler     *Scheduler[G]
	GroupSlice    []G
	stepSlice     []*Step[G]
	StepIndex     uint16
	resultFunctor func(*Sequence[G], bool, bool)
}

func NewSequence[G comparable](
	scheduler *Scheduler[G],
	groupSlice []G,
	stepSlice []*Step[G],
	resultFunctor func(*Sequence[G], bool, bool),
) *Sequence[G] {
	return &Sequence[G]{
		scheduler:     scheduler,
		GroupSlice:    groupSlice,
		stepSlice:     stepSlice,
		StepIndex:     0,
		resultFunctor: resultFunctor,
	}
}

func (s *Sequence[G]) result(stepResult bool) {
	if s.resultFunctor == nil {
		return
	}

	var sequenceResult bool
	if stepResult && s.StepIndex+1 >= uint16(len(s.stepSlice)) {
		sequenceResult = true
	}

	if s.scheduler.Options().LogDebug {
		log.Printf(
			"%s: invoking result group=%+v, stepIndex=%d, stepResult=%t, sequenceResult=%t",
			s.scheduler.Options().LogPrefix,
			s.GroupSlice,
			s.StepIndex,
			stepResult,
			sequenceResult,
		)
	}

	func() {
		defer func() {
			rec := recover()
			if rec != nil {
				log.Printf(
					"%s: group=%+v, stepIndex=%d, stepResult=%t, sequenceResult=%t, result functor recovered from panic: %+v",
					s.scheduler.Options().LogPrefix,
					s.GroupSlice,
					s.StepIndex,
					stepResult,
					sequenceResult,
					rec,
				)
			}
		}()
		s.resultFunctor(s, stepResult, sequenceResult)
	}()
}

func (s *Sequence[G]) step() {
	stepLen := uint16(len(s.stepSlice))

	for {
		if s.StepIndex >= stepLen {
			return
		}

		immediate, functor := s.stepSlice[s.StepIndex].resolve(s)

		log.Printf(
			"%s: sequence group=%+v, step<%d/%d>, immediate=%t",
			s.scheduler.Options().LogPrefix,
			s.GroupSlice,
			s.StepIndex+1,
			stepLen,
			immediate,
		)

		if functor != nil {
			func() {
				defer func() {
					rec := recover()
					if rec != nil {
						log.Printf(
							"%s: group=%+v, step<%d/%d>, immediate=%t, functor recovered from panic: %+v",
							s.scheduler.Options().LogPrefix,
							s.GroupSlice,
							s.StepIndex+1,
							stepLen,
							immediate,
							rec,
						)
					}
				}()
				functor()
			}()
		}

		if immediate {
			// step has completed
			s.result(true)

			// advance to next step synchronously
			s.StepIndex += 1
			continue
		} else {
			// async variant assumed scheduled, defer to callback via processLoop
			break
		}
	}
}

func ActionStep[G comparable](f func()) *Step[G] {
	return &Step[G]{
		resolve: func(_ *Sequence[G]) (bool, func()) {
			return true, f
		},
	}
}

func TimerStep[G comparable](d time.Duration) *Step[G] {
	return &Step[G]{
		resolve: func(s *Sequence[G]) (bool, func()) {
			return false, func() {
				timer := time.NewTimer(d)
				v := NewAsyncVariant[G](
					s.GroupSlice,
					timer.C,
					func(scheduler *Scheduler[G], v *AsyncVariant[G], _ interface{}) {
						// step has completed
						s.result(true)
						scheduler.removeAsyncVariant(v)

						// advance to next step in sequence
						s.StepIndex += 1
						s.step()
					},
					func(_ *Scheduler[G], v *AsyncVariant[G]) {
						if v.SelectCount > 0 {
							// implies timer has triggered and selected
							return
						}

						timer.Stop()
						select {
						case <-timer.C:
						default:
						}

						// notify caller sequence has been interrupted
						s.result(false)
					},
				)
				s.scheduler.addAsyncVariant(v)
			}
		},
	}
}
