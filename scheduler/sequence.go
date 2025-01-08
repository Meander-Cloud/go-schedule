package scheduler

import (
	"fmt"
	"log"
	"time"
)

type Step[G comparable] struct {
	// sequence containing this step, associated at runtime
	q *Sequence[G]

	// type of step
	stepType StepType

	// functor to invoke at each rep
	repFunctor func(*Step[G]) (bool, error)

	// total reps, zero rep will be treated as one rep
	repTotal uint16

	// reps completed
	repCount uint16
}

type Sequence[G comparable] struct {
	// associated scheduler processing this sequence
	s *Scheduler[G]

	// parent step this sequence is associated with, if any
	pp *Step[G]

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

	// progress logging mode
	logProgressMode LogProgressMode
}

func NewSequence[G comparable](
	s *Scheduler[G],
	releaseGroup bool,
	groupSlice []G,
	stepSlice []*Step[G],
	resultFunctor func(*Sequence[G], bool, bool),
	logProgressMode LogProgressMode,
) *Sequence[G] {
	return &Sequence[G]{
		s:               s,
		pp:              nil,
		releaseGroup:    releaseGroup,
		GroupSlice:      groupSlice,
		stepSlice:       stepSlice,
		StepIndex:       0,
		resultFunctor:   resultFunctor,
		logProgressMode: logProgressMode,
	}
}

func ActionStep[G comparable](f func() error) *Step[G] {
	return &Step[G]{
		q:        nil,
		stepType: StepTypeAction,
		repFunctor: func(*Step[G]) (bool, error) {
			return true, f()
		},
		repTotal: 1,
		repCount: 0,
	}
}

func TimerStep[G comparable](d time.Duration) *Step[G] {
	return &Step[G]{
		q:        nil,
		stepType: StepTypeTimer,
		repFunctor: func(p *Step[G]) (bool, error) {
			timer := time.NewTimer(d)

			v := NewAsyncVariant[G](
				false, // group release is already done optionally before start of sequence
				p.q.GroupSlice,
				timer.C,
				func(s *Scheduler[G], v *AsyncVariant[G], _ interface{}) {
					// remove and release triggered timer
					s.removeAsyncVariant(v)()

					// proceed with flow
					p.asyncRepDone()
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

					// proceed with flow
					p.asyncRepFail()
				},
			)
			p.q.s.addAsyncVariant(v)

			return false, nil
		},
		repTotal: 1,
		repCount: 0,
	}
}

func SequenceStep[G comparable](
	repTotal uint16, // zero rep will be treated as one rep
	q *Sequence[G],
) *Step[G] {
	return &Step[G]{
		q:        nil, // note that this field stores containing sequence, not input sequence which comprise this step
		stepType: StepTypeSequence,
		repFunctor: func(p *Step[G]) (bool, error) {
			return q.enter(p)
		},
		repTotal: repTotal,
		repCount: 0,
	}
}

func (p *Step[G]) take(q *Sequence[G]) (bool, error) {
	// initialize step
	p.q = q
	p.repCount = 0

	return p.rep(true)
}

func (p *Step[G]) asyncRepDone() {
	// rep done
	p.repCount += 1

	if p.repCount >= p.repTotal {
		// all reps done, up call to advance sequence
		p.q.asyncStepDone()
		return
	}

	// next rep
	p.rep(false)
}

func (p *Step[G]) asyncRepFail() {
	// up call to interrupt sequence
	p.q.asyncStepFail()
}

func (p *Step[G]) rep(inSyncLoop bool) (bool, error) {
	for {
		if p.q.logProgressMode == LogProgressModeRep {
			log.Printf(
				"%s: group=%+v, step<%d/%d>, type=%s, rep<%d/%d>",
				p.q.s.options.LogPrefix,
				p.q.GroupSlice,
				p.q.StepIndex+1,
				len(p.q.stepSlice),
				p.stepType,
				p.repCount+1,
				p.repTotal,
			)
		}

		var sync bool
		var err error
		func() {
			defer func() {
				rec := recover()
				if rec != nil {
					// panic, synchronous error
					sync = true
					err = fmt.Errorf(
						"group=%+v, step<%d/%d>, type=%s, rep<%d/%d>, functor recovered from panic: %+v",
						p.q.GroupSlice,
						p.q.StepIndex+1,
						len(p.q.stepSlice),
						p.stepType,
						p.repCount+1,
						p.repTotal,
						rec,
					)
					log.Printf("%s: %s", p.q.s.options.LogPrefix, err.Error())
				}
			}()
			sync, err = p.repFunctor(p)
		}()

		if sync {
			if err != nil {
				// rep failed
				if inSyncLoop {
					// rewind stack for caller to interrupt sequence synchronously
					return true, err
				} else {
					// call stack not in synchronous loop, up call to interrupt sequence
					p.q.asyncStepFail()
					return false, err
				}
			}

			// rep done
			p.repCount += 1

			if p.repCount >= p.repTotal {
				// all reps done
				break
			} else {
				// next rep
				continue
			}
		} else {
			// no need to check error in asynchronous flow, control will ensue via callback
			return false, nil
		}
	}

	if inSyncLoop {
		// rewind stack for caller to advance sequence synchronously
		return true, nil
	} else {
		// call stack not in synchronous loop, up call to advance sequence
		p.q.asyncStepDone()
		return false, nil
	}
}

func (q *Sequence[G]) enter(pp *Step[G]) (bool, error) {
	// initialize sequence
	q.pp = pp
	q.StepIndex = 0

	if q.releaseGroup {
		q.s.releaseGroupSlice(q.GroupSlice)
	}

	if len(q.stepSlice) == 0 {
		// proceed as success
		q.result(true)

		// rewind stack for caller to advance parent step (if any) synchronously
		return true, nil
	}

	return q.step(true)
}

func (q *Sequence[G]) asyncStepDone() {
	// step completed
	q.result(true)

	// advance step
	q.StepIndex += 1

	stepLen := uint16(len(q.stepSlice))
	if q.StepIndex >= stepLen {
		// all steps in sequence completed
		if q.pp != nil {
			// up call to advance parent step, if any
			q.pp.asyncRepDone()
		}
		return
	}

	q.step(false)
}

func (q *Sequence[G]) asyncStepFail() {
	// step failed
	q.result(false)

	if q.pp != nil {
		// up call to interrupt parent step, if any
		q.pp.asyncRepFail()
	}
}

func (q *Sequence[G]) step(inSyncLoop bool) (bool, error) {
	stepLen := uint16(len(q.stepSlice))

	for {
		p := q.stepSlice[q.StepIndex]

		if q.logProgressMode == LogProgressModeStep {
			log.Printf(
				"%s: group=%+v, step<%d/%d>, type=%s",
				q.s.options.LogPrefix,
				q.GroupSlice,
				q.StepIndex+1,
				stepLen,
				p.stepType,
			)
		}

		sync, err := p.take(q)

		if sync {
			if err != nil {
				// step failed
				q.result(false)

				if inSyncLoop {
					// rewind stack for caller to interrupt parent step (if any) synchronously
					return true, err
				} else {
					// call stack not in synchronous loop, up call to interrupt parent step, if any
					if q.pp != nil {
						q.pp.asyncRepFail()
					}
					return false, err
				}
			}

			// step completed
			q.result(true)

			// advance step
			q.StepIndex += 1

			if q.StepIndex >= stepLen {
				// all steps in sequence completed
				break
			} else {
				continue
			}
		} else {
			// no need to check error in asynchronous flow, control will ensue via callback
			return false, nil
		}
	}

	if inSyncLoop {
		// rewind stack for caller to advance parent step (if any) synchronously
		return true, nil
	} else {
		// call stack not in synchronous loop, up call to advance parent step, if any
		if q.pp != nil {
			q.pp.asyncRepDone()
		}
		return false, nil
	}
}

func (q *Sequence[G]) result(stepResult bool) {
	if q.resultFunctor == nil {
		return
	}

	sequenceResult := false
	if stepResult && q.StepIndex+1 >= uint16(len(q.stepSlice)) {
		sequenceResult = true
	}

	if q.s.options.LogDebug {
		log.Printf(
			"%s: invoking result group=%+v, stepIndex=%d, stepResult=%t, sequenceResult=%t",
			q.s.options.LogPrefix,
			q.GroupSlice,
			q.StepIndex,
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
					q.s.options.LogPrefix,
					q.GroupSlice,
					q.StepIndex,
					stepResult,
					sequenceResult,
					rec,
				)
			}
		}()
		q.resultFunctor(q, stepResult, sequenceResult)
	}()
}
