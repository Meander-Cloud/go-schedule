package scheduler

import (
	"fmt"
	"log"
	"time"
)

type Step[G comparable] struct {
	// resolve step details for invocation
	resolve func(*Sequence[G]) (bool, func() error)
}

type Sequence[G comparable] struct {
	// associated scheduler processing this sequence
	scheduler *Scheduler[G]

	// whether to release groups first when entering this sequence
	releaseGroup bool

	// groups to which this sequence belongs
	GroupSlice []G

	// steps of which this sequence is consisted
	stepSlice []*Step[G]

	// current step
	StepIndex uint16

	// functor to invoke after each step to convey step result and sequence result
	resultFunctor func(*Sequence[G], bool, bool)

	// for chaining sequence of sequences
	chainFunctor func(bool)
}

func NewSequence[G comparable](
	scheduler *Scheduler[G],
	releaseGroup bool,
	groupSlice []G,
	stepSlice []*Step[G],
	resultFunctor func(*Sequence[G], bool, bool),
) *Sequence[G] {
	return &Sequence[G]{
		scheduler:     scheduler,
		releaseGroup:  releaseGroup,
		GroupSlice:    groupSlice,
		stepSlice:     stepSlice,
		StepIndex:     0,
		resultFunctor: resultFunctor,
		chainFunctor:  nil,
	}
}

func (s *Sequence[G]) result(stepResult bool) {
	sequenceResult := false
	if stepResult && s.StepIndex+1 >= uint16(len(s.stepSlice)) {
		sequenceResult = true
	}

	defer func() {
		if s.chainFunctor == nil {
			return
		}

		if !stepResult {
			// sequence interrupted
			s.chainFunctor(false)
			return
		}

		if sequenceResult {
			// sequence completed
			s.chainFunctor(true)
			return
		}

		// sequence ongoing
	}()

	if s.resultFunctor == nil {
		return
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

func (s *Sequence[G]) enter() error {
	if s.releaseGroup {
		s.scheduler.releaseGroupSlice(s.GroupSlice)
	}

	return s.step()
}

func (s *Sequence[G]) step() error {
	stepLen := uint16(len(s.stepSlice))

	for {
		if s.StepIndex >= stepLen {
			return nil
		}

		sync, functor := s.stepSlice[s.StepIndex].resolve(s)

		log.Printf(
			"%s: sequence group=%+v, step<%d/%d>, sync=%t",
			s.scheduler.Options().LogPrefix,
			s.GroupSlice,
			s.StepIndex+1,
			stepLen,
			sync,
		)

		if functor != nil {
			var err error
			func() {
				defer func() {
					rec := recover()
					if rec != nil {
						err = fmt.Errorf(
							"group=%+v, step<%d/%d>, sync=%t, functor recovered from panic: %+v",
							s.GroupSlice,
							s.StepIndex+1,
							stepLen,
							sync,
							rec,
						)

						log.Printf("%s: %s", s.scheduler.Options().LogPrefix, err.Error())
					}
				}()
				err = functor()
			}()
			if err != nil {
				// step has failed, sequence interrupted
				s.result(false)

				return err
			}
		}

		if sync {
			// step has completed
			s.result(true)

			// advance to next step synchronously
			s.StepIndex += 1
			continue
		} else {
			// either async variant has been scheduled, or we are entering a child sequence
			// defer to callback via processLoop
			break
		}
	}

	return nil
}

func ActionStep[G comparable](f func() error) *Step[G] {
	return &Step[G]{
		resolve: func(_ *Sequence[G]) (bool, func() error) {
			return true, f
		},
	}
}

func TimerStep[G comparable](d time.Duration) *Step[G] {
	return &Step[G]{
		resolve: func(s *Sequence[G]) (bool, func() error) {
			return false, func() error {
				timer := time.NewTimer(d)
				v := NewAsyncVariant[G](
					false, // group release is already done optionally before start of sequence
					s.GroupSlice,
					timer.C,
					func(scheduler *Scheduler[G], v *AsyncVariant[G], _ interface{}) {
						// first remove triggered timer
						f := scheduler.removeAsyncVariant(v)

						// step has completed
						s.result(true)

						// invoke release
						f()

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
				return nil
			}
		},
	}
}

func SequenceStep[G comparable](this *Sequence[G]) *Step[G] {
	return &Step[G]{
		resolve: func(parent *Sequence[G]) (bool, func() error) {
			this.chainFunctor = func(sequenceResult bool) {
				if sequenceResult {
					// advance parent sequence
					parent.StepIndex += 1
					parent.step()
				} else {
					// interrupt parent sequence
					parent.result(false)
				}
			}

			return false, // break parent step loop
				this.enter // enter child step loop
		},
	}
}
