package scheduler

import (
	"log"
	"time"
)

type AsyncVariant[G comparable] struct {
	// whether to release groups first before scheduling
	releaseGroup bool

	// whether this async variant is being removed
	inRemove bool

	// handle to which this async variant is referred, generated via asyncHandleTree
	asyncHandle uint16

	// index linking to selectIndexTree, note this may change during runtime
	selectIndex uint16

	// groups to which this async variant belongs
	GroupSlice []G

	// channel to register for dynamic select
	ch interface{}

	// number of times this async variant had been selected
	SelectCount uint32

	// functor to invoke upon select
	selectFunctor func(*Scheduler[G], *AsyncVariant[G], interface{})

	// functor to invoke to release associated resources
	releaseFunctor func(*Scheduler[G], *AsyncVariant[G])
}

func NewAsyncVariant[G comparable](
	releaseGroup bool,
	groupSlice []G,
	ch interface{},
	selectFunctor func(*Scheduler[G], *AsyncVariant[G], interface{}),
	releaseFunctor func(*Scheduler[G], *AsyncVariant[G]),
) *AsyncVariant[G] {
	return &AsyncVariant[G]{
		releaseGroup: releaseGroup,
		inRemove:     false,

		// populated in addAsyncVariant
		asyncHandle: 0,
		selectIndex: 0,

		GroupSlice:     groupSlice,
		ch:             ch,
		SelectCount:    0,
		selectFunctor:  selectFunctor,
		releaseFunctor: releaseFunctor,
	}
}

func TimerAsync[G comparable](
	releaseGroup bool,
	groupSlice []G,
	d time.Duration,
	selectFunctor func(),
	releaseFunctor func(uint32),
) *AsyncVariant[G] {
	timer := time.NewTimer(d)
	return NewAsyncVariant[G](
		releaseGroup,
		groupSlice,
		timer.C,
		func(s *Scheduler[G], v *AsyncVariant[G], _ interface{}) {
			// first remove triggered timer
			f := s.removeAsyncVariant(v)

			if selectFunctor != nil {
				func() {
					defer func() {
						rec := recover()
						if rec != nil {
							log.Printf(
								"%s: handle=%d, index=%d, count=%d, group=%+v, user select functor recovered from panic: %+v",
								s.options.LogPrefix,
								v.asyncHandle,
								v.selectIndex,
								v.SelectCount,
								v.GroupSlice,
								rec,
							)
						}
					}()
					selectFunctor()
				}()
			}

			// invoke release
			f()
		},
		func(s *Scheduler[G], v *AsyncVariant[G]) {
			if v.SelectCount == 0 {
				timer.Stop()
				select {
				case <-timer.C:
				default:
				}
			}

			if releaseFunctor != nil {
				func() {
					defer func() {
						rec := recover()
						if rec != nil {
							log.Printf(
								"%s: handle=%d, index=%d, count=%d, group=%+v, user release functor recovered from panic: %+v",
								s.options.LogPrefix,
								v.asyncHandle,
								v.selectIndex,
								v.SelectCount,
								v.GroupSlice,
								rec,
							)
						}
					}()
					releaseFunctor(v.SelectCount)
				}()
			}
		},
	)
}

func TickerAsync[G comparable](
	releaseGroup bool,
	groupSlice []G,
	d time.Duration,
	selectFunctor func(),
	releaseFunctor func(uint32),
) *AsyncVariant[G] {
	ticker := time.NewTicker(d)
	return NewAsyncVariant[G](
		releaseGroup,
		groupSlice,
		ticker.C,
		func(s *Scheduler[G], v *AsyncVariant[G], _ interface{}) {
			if selectFunctor != nil {
				func() {
					defer func() {
						rec := recover()
						if rec != nil {
							log.Printf(
								"%s: handle=%d, index=%d, count=%d, group=%+v, user select functor recovered from panic: %+v",
								s.options.LogPrefix,
								v.asyncHandle,
								v.selectIndex,
								v.SelectCount,
								v.GroupSlice,
								rec,
							)
						}
					}()
					selectFunctor()
				}()
			}
		},
		func(s *Scheduler[G], v *AsyncVariant[G]) {
			ticker.Stop()
			select {
			case <-ticker.C:
			default:
			}

			if releaseFunctor != nil {
				func() {
					defer func() {
						rec := recover()
						if rec != nil {
							log.Printf(
								"%s: handle=%d, index=%d, count=%d, group=%+v, user release functor recovered from panic: %+v",
								s.options.LogPrefix,
								v.asyncHandle,
								v.selectIndex,
								v.SelectCount,
								v.GroupSlice,
								rec,
							)
						}
					}()
					releaseFunctor(v.SelectCount)
				}()
			}
		},
	)
}
