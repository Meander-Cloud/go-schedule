package scheduler

import (
	"log"
	"reflect"
	"slices"
	"sync"

	rbt "github.com/emirpasic/gods/v2/trees/redblacktree"

	"github.com/Meander-Cloud/go-chdyn/chdyn"
)

type Options struct {
	// logging prefix
	LogPrefix string

	// enable verbose logging
	LogDebug bool
}

type Scheduler[G comparable] struct {
	options *Options

	exitwg  sync.WaitGroup
	eventch *chdyn.Chan[Event]

	// group -> context
	groupContextMap map[G]*GroupContext[G]
	// async handle -> async variant
	asyncHandleTree *rbt.Tree[uint16, *AsyncVariant[G]]
	// select index -> async variant
	selectIndexTree *rbt.Tree[uint16, *AsyncVariant[G]]
	// select index -> select case
	selectCaseSlice []reflect.SelectCase
}

func NewScheduler[G comparable](options *Options) *Scheduler[G] {
	return &Scheduler[G]{
		options: options,
		exitwg:  sync.WaitGroup{},
		eventch: chdyn.New(
			&chdyn.Options[Event]{
				InSize:    chdyn.InSize,
				OutSize:   chdyn.OutSize,
				LogPrefix: options.LogPrefix,
				LogDebug:  options.LogDebug,
			},
		),
		groupContextMap: make(map[G]*GroupContext[G]),
		asyncHandleTree: rbt.New[uint16, *AsyncVariant[G]](),
		selectIndexTree: rbt.New[uint16, *AsyncVariant[G]](),
		selectCaseSlice: make([]reflect.SelectCase, 0, 256),
	}
}

func (s *Scheduler[G]) Shutdown() {
	log.Printf("%s: synchronized shutdown starting", s.options.LogPrefix)

	select {
	case s.eventch.In() <- &exitEvent{}:
	default:
		log.Printf("%s: exit already signaled", s.options.LogPrefix)
	}

	s.exitwg.Wait()
	s.eventch.Stop()
	log.Printf("%s: synchronized shutdown done", s.options.LogPrefix)
}

func (s *Scheduler[G]) Options() *Options {
	return s.options
}

func (s *Scheduler[G]) removeAsyncVariant(v *AsyncVariant[G]) func() {
	handle := v.asyncHandle
	index := v.selectIndex

	if v.inRemove {
		log.Printf(
			"%s<var>: group=%s, handle=%d, index=%d, count=%d, already removed",
			s.options.LogPrefix,
			v.PrintGroup(),
			handle,
			index,
			v.selectCount,
		)
		return func() {}
	}
	v.inRemove = true

	f := func() {
		if v.args.ReleaseFunc == nil {
			return
		}

		if s.options.LogDebug {
			log.Printf(
				"%s<var-rel>: group=%s, handle=%d, index=%d, count=%d",
				s.options.LogPrefix,
				v.PrintGroup(),
				handle,
				index,
				v.selectCount,
			)
		}

		func() {
			defer func() {
				rec := recover()
				if rec != nil {
					log.Printf(
						"%s<var-rel>: group=%s, handle=%d, index=%d, count=%d, functor recovered from panic: %+v",
						s.options.LogPrefix,
						v.PrintGroup(),
						handle,
						index,
						v.selectCount,
						rec,
					)
				}
			}()

			v.args.ReleaseFunc(
				s,
				v,
			)
		}()
	}

	func() {
		if v.args.GroupTracker == nil {
			return
		}

		for group := range v.args.GroupTracker.uniqueGroupMap {
			context, found := s.groupContextMap[group]
			if !found {
				log.Printf(
					"%s<var>: group=%+v not found in groupContextMap",
					s.options.LogPrefix,
					group,
				)
				continue
			}

			context.asyncHandleTree.Remove(handle)

			if context.asyncHandleTree.Empty() {
				delete(s.groupContextMap, group)
			}
		}
	}()

	s.selectIndexTree.Remove(index)
	s.asyncHandleTree.Remove(handle)

	intIndex := int(index)
	intLen := len(s.selectCaseSlice)
	intLast := intLen - 1

	if intIndex < intLast {
		// swap index with the last element to minimize array shift
		swapIndex := uint16(intLast)
		swapVariant, found := s.selectIndexTree.Get(swapIndex)
		if !found {
			log.Printf(
				"%s<var>: swapIndex=%d, not found",
				s.options.LogPrefix,
				swapIndex,
			)
			return f
		}
		if swapVariant.selectIndex != swapIndex {
			log.Printf(
				"%s<var>: swapVariant.selectIndex=%d mismatch swapIndex=%d",
				s.options.LogPrefix,
				swapVariant.selectIndex,
				swapIndex,
			)
			return f
		}

		s.selectIndexTree.Remove(swapIndex)
		s.selectCaseSlice[index] = s.selectCaseSlice[swapIndex]
		swapVariant.selectIndex = index
		s.selectIndexTree.Put(index, swapVariant)
	}

	// this will also properly zero value the deleted element
	s.selectCaseSlice = slices.Delete(
		s.selectCaseSlice,
		intLast,
		intLen,
	)

	if s.options.LogDebug {
		log.Printf(
			"%s<var-rem>: group=%s, handle=%d, index=%d, count=%d",
			s.options.LogPrefix,
			v.PrintGroup(),
			handle,
			index,
			v.selectCount,
		)
	}

	return f
}

func (s *Scheduler[G]) releaseAsyncVariantByTree(scopedIndexTree *rbt.Tree[uint16, *AsyncVariant[G]]) {
	if scopedIndexTree.Empty() {
		return
	}

	// we start from high -> low index, to minimize selectCaseSlice shift
	it := scopedIndexTree.Iterator()
	it.End()
	for it.Prev() {
		s.removeAsyncVariant(it.Value())()
	}
}

func (s *Scheduler[G]) releaseAll() {
	if s.options.LogDebug {
		log.Printf(
			"%s<release>: all, size=%d",
			s.options.LogPrefix,
			s.selectIndexTree.Size(),
		)
	}

	scopedIndexTree := rbt.New[uint16, *AsyncVariant[G]]()

	it := s.selectIndexTree.Iterator()
	for it.Next() {
		scopedIndexTree.Put(
			it.Key(),
			it.Value(),
		)
	}

	s.releaseAsyncVariantByTree(scopedIndexTree)
}

func (s *Scheduler[G]) packScopedIndexTree(scopedIndexTree *rbt.Tree[uint16, *AsyncVariant[G]], group G) {
	context, found := s.groupContextMap[group]
	if !found {
		if s.options.LogDebug {
			log.Printf(
				"%s<release>: group=%+v, not found",
				s.options.LogPrefix,
				group,
			)
		}
		return
	}

	it := context.asyncHandleTree.Iterator()
	for it.Next() {
		v := it.Value()

		// note that one async variant may associate with multiple groups, here we collect unique occurrences
		scopedIndexTree.Put(
			v.selectIndex,
			v,
		)
	}
}

func (s *Scheduler[G]) releaseGroup(group G) {
	scopedIndexTree := rbt.New[uint16, *AsyncVariant[G]]()

	s.packScopedIndexTree(
		scopedIndexTree,
		group,
	)

	if s.options.LogDebug {
		log.Printf(
			"%s<release>: group=%+v, size=%d",
			s.options.LogPrefix,
			group,
			scopedIndexTree.Size(),
		)
	}

	s.releaseAsyncVariantByTree(scopedIndexTree)
}

func (s *Scheduler[G]) releaseGroupSlice(groupSlice []G) {
	scopedIndexTree := rbt.New[uint16, *AsyncVariant[G]]()

	for _, group := range groupSlice {
		s.packScopedIndexTree(
			scopedIndexTree,
			group,
		)
	}

	if s.options.LogDebug {
		log.Printf(
			"%s<release>: group=%+v, size=%d",
			s.options.LogPrefix,
			groupSlice,
			scopedIndexTree.Size(),
		)
	}

	s.releaseAsyncVariantByTree(scopedIndexTree)
}

func (s *Scheduler[G]) releaseGroupTracker(groupTracker *GroupTracker[G]) {
	if groupTracker == nil {
		return
	}

	scopedIndexTree := rbt.New[uint16, *AsyncVariant[G]]()

	for group := range groupTracker.uniqueGroupMap {
		s.packScopedIndexTree(
			scopedIndexTree,
			group,
		)
	}

	if s.options.LogDebug {
		log.Printf(
			"%s<release>: group=%s, size=%d",
			s.options.LogPrefix,
			groupTracker.Print(),
			scopedIndexTree.Size(),
		)
	}

	s.releaseAsyncVariantByTree(scopedIndexTree)
}

func (s *Scheduler[G]) addAsyncVariant(v *AsyncVariant[G]) {
	if v.args.ReleaseGroup {
		s.releaseGroupTracker(v.args.GroupTracker)
	}

	var handle, index uint16

	rightNode := s.asyncHandleTree.Right()
	if rightNode == nil {
		handle = 1
	} else {
		handle = rightNode.Key + 1
	}

	s.selectCaseSlice = append(
		s.selectCaseSlice,
		reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(v.args.Chan),
		},
	)
	index = uint16(len(s.selectCaseSlice) - 1)

	v.asyncHandle = handle
	v.selectIndex = index

	s.asyncHandleTree.Put(
		handle,
		v,
	)
	s.selectIndexTree.Put(
		index,
		v,
	)

	func() {
		if v.args.GroupTracker == nil {
			return
		}

		for group := range v.args.GroupTracker.uniqueGroupMap {
			context, found := s.groupContextMap[group]
			if !found {
				context = &GroupContext[G]{
					group:           group,
					asyncHandleTree: rbt.New[uint16, *AsyncVariant[G]](),
				}

				s.groupContextMap[group] = context
			}

			context.asyncHandleTree.Put(
				handle,
				v,
			)
		}
	}()

	if s.options.LogDebug {
		log.Printf(
			"%s<var-add>: group=%s, handle=%d, index=%d, count=%d",
			s.options.LogPrefix,
			v.PrintGroup(),
			handle,
			index,
			v.selectCount,
		)
	}
}

func (s *Scheduler[G]) scheduleAsyncEvent(event *ScheduleAsyncEvent[G]) {
	s.addAsyncVariant(event.AsyncVariant)
}

func (s *Scheduler[G]) scheduleSequenceEvent(event *ScheduleSequenceEvent[G]) {
	event.Sequence.enter(nil)
}

func (s *Scheduler[G]) handle(recv interface{}) bool {
	switch event := recv.(type) {
	case *exitEvent:
		s.releaseAll()
		return true
	case *ReleaseGroupEvent[G]:
		s.releaseGroup(event.Group)
	case *ReleaseGroupSliceEvent[G]:
		s.releaseGroupSlice(event.GroupSlice)
	case *ScheduleAsyncEvent[G]:
		s.scheduleAsyncEvent(event)
	case *ScheduleSequenceEvent[G]:
		s.scheduleSequenceEvent(event)
	default:
		log.Printf(
			"%s<handle>: unrecognized recv=%#v",
			s.options.LogPrefix,
			recv,
		)
	}
	return false
}

// main event processing loop, will block;
// caller can choose to run synchronously on caller goroutine, or spawn a separate goroutine to run asynchronously
func (s *Scheduler[G]) processLoop() {
	// zero index must be case eventch
	s.selectCaseSlice = append(
		s.selectCaseSlice,
		reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(s.eventch.Out()),
		},
	)

labelFor:
	for {
		if s.options.LogDebug {
			log.Printf(
				"%s<hold>: groupContextMap<%d>, asyncHandleTree<%d>, selectIndexTree<%d>, selectCaseSlice<%d>",
				s.options.LogPrefix,
				len(s.groupContextMap),
				s.asyncHandleTree.Size(),
				s.selectIndexTree.Size(),
				len(s.selectCaseSlice),
			)
		}

		index, received, ok := reflect.Select(s.selectCaseSlice)

		if s.options.LogDebug {
			log.Printf(
				"%s<select>: index=%d, ok=%t",
				s.options.LogPrefix,
				index,
				ok,
			)
		}

		switch index {
		case 0: // corresponds to eventch
			if s.handle(received.Interface()) {
				break labelFor
			}
		default:
			v, found := s.selectIndexTree.Get(uint16(index))
			if found {
				v.selectCount += 1

				if v.args.SelectFunc != nil {
					if s.options.LogDebug {
						log.Printf(
							"%s<var-sel>: group=%s, handle=%d, index=%d, count=%d",
							s.options.LogPrefix,
							v.PrintGroup(),
							v.asyncHandle,
							v.selectIndex,
							v.selectCount,
						)
					}

					func() {
						defer func() {
							rec := recover()
							if rec != nil {
								log.Printf(
									"%s<var-sel>: group=%s, handle=%d, index=%d, count=%d, functor recovered from panic: %+v",
									s.options.LogPrefix,
									v.PrintGroup(),
									v.asyncHandle,
									v.selectIndex,
									v.selectCount,
									rec,
								)
							}
						}()

						v.args.SelectFunc(
							s,
							v,
							received.Interface(),
						)
					}()
				}
			} else {
				log.Printf(
					"%s<var>: index=%d, not found",
					s.options.LogPrefix,
					index,
				)
			}
		}
	}
}

func (s *Scheduler[G]) RunSync() {
	s.exitwg.Add(1)

	log.Printf("%s: synchronous process loop starting", s.options.LogPrefix)

	defer func() {
		log.Printf("%s: synchronous process loop exiting", s.options.LogPrefix)
		s.exitwg.Done()
	}()

	s.processLoop()
}

func (s *Scheduler[G]) RunAsync() {
	s.exitwg.Add(1)

	go func() {
		log.Printf("%s: asynchronous process loop starting", s.options.LogPrefix)

		defer func() {
			log.Printf("%s: asynchronous process loop exiting", s.options.LogPrefix)
			s.exitwg.Done()
		}()

		s.processLoop()
	}()
}

// must be invoked on same goroutine as processLoop
func (s *Scheduler[G]) ProcessSync(event Event) {
	s.handle(event)
}

// can be invoked on any goroutine
func (s *Scheduler[G]) ProcessAsync(event Event) {
	s.eventch.In() <- event
}
