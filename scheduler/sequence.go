package scheduler

import (
	"fmt"
	"log"
	"strings"
	"time"
)

type StepArgs[G comparable] struct {
	// type of step
	stepType StepType

	// describes the step with details as necessary
	descriptor func() string

	// functor to invoke at each rep
	repFunc func(*Step[G]) (bool, error)

	// total reps to execute
	repTotal uint16
}

type Step[G comparable] struct {
	args *StepArgs[G]

	// parent sequence containing this step, associated at runtime
	pq *Sequence[G]

	// current rep
	repNum uint16
}

type SequenceArgs[G comparable] struct {
	// associated scheduler processing this sequence
	Scheduler *Scheduler[G]

	// whether to inherit groups from parent sequence, if any
	InheritGroup bool

	// whether to release groups first when entering this sequence
	ReleaseGroup bool

	// groups to which this sequence belongs, before inheritance
	// note: modifying post initialization would be undefined behavior
	GroupSlice []G

	// steps of which this sequence is consisted
	StepSlice []*Step[G]

	// functor to invoke after each step to convey step result
	StepResultFunc func(*Step[G], bool)

	// functor to invoke after completion or interruption of sequence
	SequenceResultFunc func(*Sequence[G], bool)

	// progress logging mode
	LogProgressMode LogProgressMode
}

type Sequence[G comparable] struct {
	args *SequenceArgs[G]

	// parent step containing this sequence if any, associated at runtime
	pp *Step[G]

	// overall tracking of groups, computed at runtime
	groupTracker *GroupTracker[G]

	// current step
	stepNum uint16
}

func newStep[G comparable](args *StepArgs[G]) *Step[G] {
	if args == nil {
		log.Printf("Step: nil args")
		return nil
	}

	return &Step[G]{
		args:   args,
		pq:     nil,
		repNum: 0,
	}
}

type ActionStepArgs[G comparable] struct {
	Action func() error
}

func ActionStep[G comparable](args *ActionStepArgs[G]) *Step[G] {
	if args == nil {
		log.Printf("ActionStep: nil args")
		return nil
	}

	descriptorCache := ""

	return newStep(
		&StepArgs[G]{
			stepType: StepTypeAction,
			descriptor: func() string {
				if descriptorCache != "" {
					return descriptorCache
				}

				descriptorCache = StepTypeAction.Abbr()
				return descriptorCache
			},
			repFunc: func(*Step[G]) (bool, error) {
				return true, args.Action()
			},
			repTotal: 1,
		},
	)
}

type TimerStepArgs[G comparable] struct {
	Delay time.Duration
}

func TimerStep[G comparable](args *TimerStepArgs[G]) *Step[G] {
	if args == nil {
		log.Printf("TimerStep: nil args")
		return nil
	}

	descriptorCache := ""

	return newStep(
		&StepArgs[G]{
			stepType: StepTypeTimer,
			descriptor: func() string {
				if descriptorCache != "" {
					return descriptorCache
				}

				descriptorCache = fmt.Sprintf(
					"%s<%v>",
					StepTypeTimer.Abbr(),
					args.Delay,
				)
				return descriptorCache
			},
			repFunc: func(p *Step[G]) (bool, error) {
				timer := time.NewTimer(args.Delay)

				v := NewAsyncVariant(
					&AsyncVariantArgs[G]{
						ReleaseGroup: false, // group release already done optionally before start of sequence
						GroupTracker: p.pq.groupTracker,
						Chan:         timer.C,
						SelectFunc: func(s *Scheduler[G], v *AsyncVariant[G], _ interface{}) {
							// remove and release triggered timer
							s.removeAsyncVariant(v)()

							// proceed with flow
							p.asyncRepDone()
						},
						ReleaseFunc: func(_ *Scheduler[G], v *AsyncVariant[G]) {
							if v.selectCount > 0 {
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
					},
				)
				p.pq.args.Scheduler.addAsyncVariant(v)

				return false, nil
			},
			repTotal: 1,
		},
	)
}

type SequenceStepArgs[G comparable] struct {
	Sequence *Sequence[G] // input sequence which comprise this step, not step's containing sequence
	RepTotal uint16
}

func SequenceStep[G comparable](args *SequenceStepArgs[G]) *Step[G] {
	if args == nil {
		log.Printf("SequenceStep: nil args")
		return nil
	}

	descriptorCache := ""

	return newStep(
		&StepArgs[G]{
			stepType: StepTypeSequence,
			descriptor: func() string {
				if descriptorCache != "" {
					return descriptorCache
				}

				sb := new(strings.Builder)
				sb.Grow(16)
				sb.WriteString(StepTypeSequence.Abbr())
				if args.RepTotal > 1 {
					fmt.Fprintf(
						sb,
						"<x%d>",
						args.RepTotal,
					)
				}

				descriptorCache = sb.String()
				return descriptorCache
			},
			repFunc: func(p *Step[G]) (bool, error) {
				return args.Sequence.enter(p)
			},
			repTotal: args.RepTotal,
		},
	)
}

func (p *Step[G]) StepType() StepType {
	return p.args.stepType
}

func (p *Step[G]) Descriptor() string {
	return p.args.descriptor()
}

func (p *Step[G]) RepTotal() uint16 {
	return p.args.repTotal
}

func (p *Step[G]) RepNum() uint16 {
	return p.repNum
}

func (p *Step[G]) PrintGroup() string {
	if p.pq == nil {
		return ""
	}

	return p.pq.PrintGroup()
}

func (p *Step[G]) StepLen() uint16 {
	if p.pq == nil {
		return 0
	}

	return p.pq.StepLen()
}

func (p *Step[G]) StepNum() uint16 {
	if p.pq == nil {
		return 0
	}

	return p.pq.StepNum()
}

func (p *Step[G]) reset() {
	p.pq = nil
	p.repNum = 0
}

func (p *Step[G]) take(pq *Sequence[G]) (bool, error) {
	// initialize step
	p.pq = pq

	if p.args.repTotal == 0 {
		// proceed as success
		p.result(true)

		// rewind stack for caller to advance parent sequence synchronously
		p.reset()
		return true, nil
	}

	p.repNum = 1
	return p.rep(true)
}

func (p *Step[G]) asyncRepDone() {
	// rep completed
	p.result(true)

	// advance rep
	p.repNum += 1

	if p.repNum > p.args.repTotal {
		// all reps completed, up call to advance parent sequence
		pq := p.pq
		p.reset()
		pq.asyncStepDone()
		return
	}

	p.rep(false)
}

func (p *Step[G]) asyncRepFail() {
	// rep failed
	p.result(false)

	// up call to interrupt parent sequence
	pq := p.pq
	p.reset()
	pq.asyncStepFail()
}

func (p *Step[G]) rep(inSyncLoop bool) (bool, error) {
	for {
		switch p.pq.args.LogProgressMode {
		case LogProgressModeRep:
			log.Printf(
				"%s<seq>: group=%s, step<%d/%d>, type=%s, rep<%d/%d>",
				p.pq.args.Scheduler.options.LogPrefix,
				p.pq.PrintGroup(),
				p.pq.stepNum,
				len(p.pq.args.StepSlice),
				p.args.descriptor(),
				p.repNum,
				p.args.repTotal,
			)
		default:
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
						"%s<seq-rep>: group=%s, step<%d/%d>, type=%s, rep<%d/%d>, functor recovered from panic: %+v",
						p.pq.args.Scheduler.options.LogPrefix,
						p.pq.PrintGroup(),
						p.pq.stepNum,
						len(p.pq.args.StepSlice),
						p.args.descriptor(),
						p.repNum,
						p.args.repTotal,
						rec,
					)
					log.Printf("%s", err.Error())
				}
			}()

			sync, err = p.args.repFunc(p)
		}()

		if sync {
			if err != nil {
				// rep failed
				p.result(false)

				if inSyncLoop {
					// rewind stack for caller to interrupt parent sequence synchronously
					p.reset()
					return true, err
				} else {
					// call stack not in synchronous loop, up call to interrupt parent sequence
					pq := p.pq
					p.reset()
					pq.asyncStepFail()
					return false, err
				}
			}

			// rep completed
			p.result(true)

			// advance rep
			p.repNum += 1

			if p.repNum > p.args.repTotal {
				// all reps completed
				break
			} else {
				continue
			}
		} else {
			// no need to check error in asynchronous flow, control will ensue via callback
			return false, nil
		}
	}

	// all reps completed
	if inSyncLoop {
		// rewind stack for caller to advance parent sequence synchronously
		p.reset()
		return true, nil
	} else {
		// call stack not in synchronous loop, up call to advance parent sequence
		pq := p.pq
		p.reset()
		pq.asyncStepDone()
		return false, nil
	}
}

func (p *Step[G]) result(repResult bool) {
	switch p.pq.args.LogProgressMode {
	case LogProgressModeRep:
		log.Printf(
			"%s<seq-res>: group=%s, step<%d/%d>, type=%s, rep<%d/%d>, result=%t",
			p.pq.args.Scheduler.options.LogPrefix,
			p.pq.PrintGroup(),
			p.pq.stepNum,
			len(p.pq.args.StepSlice),
			p.args.descriptor(),
			p.repNum,
			p.args.repTotal,
			repResult,
		)
	default:
	}

	func() {
		if p.pq.args.StepResultFunc == nil {
			return
		}

		// invoke functor if any rep fails, or if all reps have completed
		if repResult && p.repNum < p.args.repTotal {
			return
		}

		defer func() {
			rec := recover()
			if rec != nil {
				log.Printf(
					"%s<seq-step-res>: group=%s, step<%d/%d>, type=%s, rep<%d/%d>, result=%t, functor recovered from panic: %+v",
					p.pq.args.Scheduler.options.LogPrefix,
					p.pq.PrintGroup(),
					p.pq.stepNum,
					len(p.pq.args.StepSlice),
					p.args.descriptor(),
					p.repNum,
					p.args.repTotal,
					repResult,
					rec,
				)
			}
		}()

		p.pq.args.StepResultFunc(
			p,
			repResult,
		)
	}()
}

func NewSequence[G comparable](args *SequenceArgs[G]) *Sequence[G] {
	if args == nil {
		log.Printf("Sequence: nil args")
		return nil
	}

	return &Sequence[G]{
		args:         args,
		pp:           nil,
		groupTracker: nil,
		stepNum:      0,
	}
}

func (q *Sequence[G]) PrintGroup() string {
	return q.groupTracker.Print()
}

func (q *Sequence[G]) StepLen() uint16 {
	return uint16(len(q.args.StepSlice))
}

func (q *Sequence[G]) StepNum() uint16 {
	return q.stepNum
}

func (q *Sequence[G]) reset() {
	q.pp = nil
	q.groupTracker = nil
	q.stepNum = 0
}

func (q *Sequence[G]) enter(pp *Step[G]) (bool, error) {
	// initialize sequence
	q.pp = pp
	if q.args.InheritGroup &&
		pp != nil {
		q.groupTracker = newGroupTrackerWithParent(
			pp.pq.groupTracker,
			q.args.GroupSlice,
		)
	} else {
		q.groupTracker = NewGroupTracker(
			q.args.GroupSlice,
		)
	}

	if q.args.ReleaseGroup {
		q.args.Scheduler.releaseGroupTracker(q.groupTracker)
	}

	if len(q.args.StepSlice) == 0 {
		// proceed as success
		q.result(true)

		// rewind stack for caller to advance parent step synchronously, if any
		q.reset()
		return true, nil
	}

	q.stepNum = 1
	return q.step(true)
}

func (q *Sequence[G]) asyncStepDone() {
	// step completed
	q.result(true)

	// advance step
	q.stepNum += 1

	stepLen := uint16(len(q.args.StepSlice))
	if q.stepNum > stepLen {
		// all steps completed
		pp := q.pp
		q.reset()
		if pp != nil {
			// up call to advance parent step, if any
			pp.asyncRepDone()
		}
		return
	}

	q.step(false)
}

func (q *Sequence[G]) asyncStepFail() {
	// step failed
	q.result(false)

	pp := q.pp
	q.reset()
	if pp != nil {
		// up call to interrupt parent step, if any
		pp.asyncRepFail()
	}
}

func (q *Sequence[G]) step(inSyncLoop bool) (bool, error) {
	stepLen := uint16(len(q.args.StepSlice))

	for {
		// note that stepNum starts at 1
		p := q.args.StepSlice[q.stepNum-1]

		switch q.args.LogProgressMode {
		case LogProgressModeStep:
			log.Printf(
				"%s<seq>: group=%s, step<%d/%d>, type=%s",
				q.args.Scheduler.options.LogPrefix,
				q.PrintGroup(),
				q.stepNum,
				stepLen,
				p.args.descriptor(),
			)
		default:
		}

		sync, err := p.take(q)

		if sync {
			if err != nil {
				// step failed
				q.result(false)

				if inSyncLoop {
					// rewind stack for caller to interrupt parent step synchronously, if any
					q.reset()
					return true, err
				} else {
					// call stack not in synchronous loop, up call to interrupt parent step, if any
					pp := q.pp
					q.reset()
					if pp != nil {
						pp.asyncRepFail()
					}
					return false, err
				}
			}

			// step completed
			q.result(true)

			// advance step
			q.stepNum += 1

			if q.stepNum > stepLen {
				// all steps completed
				break
			} else {
				continue
			}
		} else {
			// no need to check error in asynchronous flow, control will ensue via callback
			return false, nil
		}
	}

	// all steps completed
	if inSyncLoop {
		// rewind stack for caller to advance parent step synchronously, if any
		q.reset()
		return true, nil
	} else {
		// call stack not in synchronous loop, up call to advance parent step, if any
		pp := q.pp
		q.reset()
		if pp != nil {
			pp.asyncRepDone()
		}
		return false, nil
	}
}

func (q *Sequence[G]) result(stepResult bool) {
	stepLen := uint16(len(q.args.StepSlice))

	var step *Step[G]
	if q.stepNum >= 1 && q.stepNum <= stepLen {
		step = q.args.StepSlice[q.stepNum-1]
	}

	switch q.args.LogProgressMode {
	case LogProgressModeStep:
		log.Printf(
			"%s<seq-res>: group=%s, step<%d/%d>, type=%s, result=%t",
			q.args.Scheduler.options.LogPrefix,
			q.PrintGroup(),
			q.stepNum,
			stepLen,
			func() string {
				if step == nil {
					return "nil"
				}
				return step.args.descriptor()
			}(),
			stepResult,
		)
	default:
	}

	func() {
		if q.args.SequenceResultFunc == nil {
			return
		}

		// invoke functor if any step fails, or if all steps have completed
		if stepResult && q.stepNum < stepLen {
			return
		}

		defer func() {
			rec := recover()
			if rec != nil {
				log.Printf(
					"%s<seq-seq-res>: group=%s, step<%d/%d>, type=%s, result=%t, functor recovered from panic: %+v",
					q.args.Scheduler.options.LogPrefix,
					q.PrintGroup(),
					q.stepNum,
					stepLen,
					func() string {
						if step == nil {
							return "nil"
						}
						return step.args.descriptor()
					}(),
					stepResult,
					rec,
				)
			}
		}()

		q.args.SequenceResultFunc(
			q,
			stepResult,
		)
	}()
}
