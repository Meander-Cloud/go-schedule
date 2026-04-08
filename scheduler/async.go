package scheduler

import (
	"log"
	"time"
)

type AsyncVariantArgs[G comparable] struct {
	// whether to release groups first before scheduling
	ReleaseGroup bool

	// groups to which this async variant belongs
	GroupTracker *GroupTracker[G]

	// channel to register for dynamic select
	Chan interface{}

	// functor to invoke upon select
	SelectFunc func(*Scheduler[G], *AsyncVariant[G], interface{})

	// functor to invoke to release associated resources
	ReleaseFunc func(*Scheduler[G], *AsyncVariant[G])
}

type AsyncVariant[G comparable] struct {
	args *AsyncVariantArgs[G]

	// whether this async variant is being removed
	inRemove bool

	// handle to which this async variant is referred, generated via asyncHandleTree
	asyncHandle uint16

	// index linking to selectIndexTree, note this may change during runtime
	selectIndex uint16

	// number of times this async variant has been selected
	selectCount uint32
}

func NewAsyncVariant[G comparable](args *AsyncVariantArgs[G]) *AsyncVariant[G] {
	if args == nil {
		log.Printf("AsyncVariant: nil args")
		return nil
	}

	return &AsyncVariant[G]{
		args:        args,
		inRemove:    false,
		asyncHandle: 0, // populated in addAsyncVariant
		selectIndex: 0, // populated in addAsyncVariant
		selectCount: 0,
	}
}

func (v *AsyncVariant[G]) PrintGroup() string {
	return v.args.GroupTracker.Print()
}

func (v *AsyncVariant[G]) SelectCount() uint32 {
	return v.selectCount
}

type TimerAsyncArgs[G comparable] struct {
	ReleaseGroup bool
	GroupSlice   []G
	Delay        time.Duration
	SelectFunc   func()
	ReleaseFunc  func(uint32)
}

func TimerAsync[G comparable](args *TimerAsyncArgs[G]) *AsyncVariant[G] {
	if args == nil {
		log.Printf("TimerAsync: nil args")
		return nil
	}

	timer := time.NewTimer(args.Delay)

	return NewAsyncVariant(
		&AsyncVariantArgs[G]{
			ReleaseGroup: args.ReleaseGroup,
			GroupTracker: NewGroupTracker(args.GroupSlice),
			Chan:         timer.C,
			SelectFunc: func(s *Scheduler[G], v *AsyncVariant[G], _ interface{}) {
				// first remove triggered timer
				f := s.removeAsyncVariant(v)

				if args.SelectFunc != nil {
					func() {
						defer func() {
							rec := recover()
							if rec != nil {
								log.Printf(
									"%s<tim-var-sel>: group=%s, handle=%d, index=%d, count=%d, functor recovered from panic: %+v",
									s.options.LogPrefix,
									v.PrintGroup(),
									v.asyncHandle,
									v.selectIndex,
									v.selectCount,
									rec,
								)
							}
						}()

						args.SelectFunc()
					}()
				}

				// invoke release
				f()
			},
			ReleaseFunc: func(s *Scheduler[G], v *AsyncVariant[G]) {
				if v.selectCount == 0 {
					timer.Stop()
					select {
					case <-timer.C:
					default:
					}
				}

				if args.ReleaseFunc != nil {
					func() {
						defer func() {
							rec := recover()
							if rec != nil {
								log.Printf(
									"%s<tim-var-rel>: group=%s, handle=%d, index=%d, count=%d, functor recovered from panic: %+v",
									s.options.LogPrefix,
									v.PrintGroup(),
									v.asyncHandle,
									v.selectIndex,
									v.selectCount,
									rec,
								)
							}
						}()

						args.ReleaseFunc(v.selectCount)
					}()
				}
			},
		},
	)
}

type TickerAsyncArgs[G comparable] struct {
	ReleaseGroup bool
	GroupSlice   []G
	Interval     time.Duration
	SelectFunc   func()
	ReleaseFunc  func(uint32)
}

func TickerAsync[G comparable](args *TickerAsyncArgs[G]) *AsyncVariant[G] {
	if args == nil {
		log.Printf("TickerAsync: nil args")
		return nil
	}

	ticker := time.NewTicker(args.Interval)

	return NewAsyncVariant(
		&AsyncVariantArgs[G]{
			ReleaseGroup: args.ReleaseGroup,
			GroupTracker: NewGroupTracker(args.GroupSlice),
			Chan:         ticker.C,
			SelectFunc: func(s *Scheduler[G], v *AsyncVariant[G], _ interface{}) {
				if args.SelectFunc != nil {
					func() {
						defer func() {
							rec := recover()
							if rec != nil {
								log.Printf(
									"%s<tic-var-sel>: group=%s, handle=%d, index=%d, count=%d, functor recovered from panic: %+v",
									s.options.LogPrefix,
									v.PrintGroup(),
									v.asyncHandle,
									v.selectIndex,
									v.selectCount,
									rec,
								)
							}
						}()

						args.SelectFunc()
					}()
				}
			},
			ReleaseFunc: func(s *Scheduler[G], v *AsyncVariant[G]) {
				ticker.Stop()
				select {
				case <-ticker.C:
				default:
				}

				if args.ReleaseFunc != nil {
					func() {
						defer func() {
							rec := recover()
							if rec != nil {
								log.Printf(
									"%s<tic-var-rel>: group=%s, handle=%d, index=%d, count=%d, functor recovered from panic: %+v",
									s.options.LogPrefix,
									v.PrintGroup(),
									v.asyncHandle,
									v.selectIndex,
									v.selectCount,
									rec,
								)
							}
						}()

						args.ReleaseFunc(v.selectCount)
					}()
				}
			},
		},
	)
}
