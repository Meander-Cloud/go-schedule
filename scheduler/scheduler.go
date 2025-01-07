package scheduler

import (
	"log"
	"reflect"
	"slices"
	"sync"

	rbt "github.com/emirpasic/gods/v2/trees/redblacktree"
)

type Options struct {
	// specify length for event channel, if zero default will be used
	EventChannelLength uint16

	// logging prefix
	LogPrefix string

	// enable verbose logging
	LogDebug bool
}

type Scheduler[G comparable] struct {
	options *Options

	exitwg  sync.WaitGroup
	eventch chan Event

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
	var eventChannelLength uint16
	if options.EventChannelLength == 0 {
		eventChannelLength = EventChannelLength
	} else {
		eventChannelLength = options.EventChannelLength
	}

	return &Scheduler[G]{
		options: options,

		exitwg:  sync.WaitGroup{},
		eventch: make(chan Event, eventChannelLength),

		groupContextMap: make(map[G]*GroupContext[G]),
		asyncHandleTree: rbt.New[uint16, *AsyncVariant[G]](),
		selectIndexTree: rbt.New[uint16, *AsyncVariant[G]](),
		selectCaseSlice: make([]reflect.SelectCase, 0, 256),
	}
}

func (s *Scheduler[G]) Shutdown() {
	log.Printf("%s: synchronized shutdown starting", s.options.LogPrefix)

	select {
	case s.eventch <- &exitEvent{}:
	default:
		log.Printf("%s: exit already signaled", s.options.LogPrefix)
	}

	s.exitwg.Wait()
	log.Printf("%s: synchronized shutdown done", s.options.LogPrefix)
}

func (s *Scheduler[G]) Options() *Options {
	return s.options
}

func (s *Scheduler[G]) RunSync() {
	s.exitwg.Add(1)
	defer s.exitwg.Done()

	log.Printf("%s: synchronous process loop starting", s.options.LogPrefix)
	s.processLoop()
	log.Printf("%s: synchronous process loop exiting", s.options.LogPrefix)
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

func (s *Scheduler[G]) processLoop() {
	// main event processing loop, will block
	// caller can choose to run synchronously on caller goroutine, or spawn a separate goroutine to run asynchronously

	// zero index must be case eventch
	s.selectCaseSlice = append(
		s.selectCaseSlice,
		reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(s.eventch),
		},
	)

labelFor:
	for {
		if s.options.LogDebug {
			log.Printf(
				"%s: groupContextMap<%d>, asyncHandleTree<%d>, selectIndexTree<%d>, selectCaseSlice<%d>",
				s.options.LogPrefix,
				len(s.groupContextMap),
				s.asyncHandleTree.Size(),
				s.selectIndexTree.Size(),
				len(s.selectCaseSlice),
			)
		}

		index, received, ok := reflect.Select(s.selectCaseSlice)

		if s.options.LogDebug {
			log.Printf("%s: selected index=%d, ok=%t", s.options.LogPrefix, index, ok)
		}

		switch index {
		case 0: // corresponds to eventch
			if s.handle(received.Interface()) {
				break labelFor
			}
		default:
			v, found := s.selectIndexTree.Get(uint16(index))
			if found {
				v.SelectCount += 1

				if v.selectFunctor != nil {
					if s.options.LogDebug {
						log.Printf(
							"%s: invoking select handle=%d, index=%d, count=%d, group=%+v",
							s.options.LogPrefix,
							v.asyncHandle,
							v.selectIndex,
							v.SelectCount,
							v.GroupSlice,
						)
					}

					func() {
						defer func() {
							rec := recover()
							if rec != nil {
								log.Printf(
									"%s: handle=%d, index=%d, count=%d, group=%+v, select functor recovered from panic: %+v",
									s.options.LogPrefix,
									v.asyncHandle,
									v.selectIndex,
									v.SelectCount,
									v.GroupSlice,
									rec,
								)
							}
						}()
						v.selectFunctor(s, v, received.Interface())
					}()
				}
			} else {
				log.Printf("%s: no async variant found for index=%d", s.options.LogPrefix, index)
			}
		}
	}
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
		log.Printf("%s: unrecognized recv=%#v", s.options.LogPrefix, recv)
	}
	return false
}

func (s *Scheduler[G]) addAsyncVariant(v *AsyncVariant[G]) {
	if v.releaseGroup {
		s.releaseGroupSlice(v.GroupSlice)
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
			Chan: reflect.ValueOf(v.ch),
		},
	)
	index = uint16(len(s.selectCaseSlice) - 1)

	v.asyncHandle = handle
	v.selectIndex = index

	s.asyncHandleTree.Put(handle, v)
	s.selectIndexTree.Put(index, v)

	for _, group := range v.GroupSlice {
		context, found := s.groupContextMap[group]
		if !found {
			context = &GroupContext[G]{
				group: group,

				asyncHandleTree: rbt.New[uint16, *AsyncVariant[G]](),
			}

			s.groupContextMap[group] = context
		}

		context.asyncHandleTree.Put(handle, v)
	}

	if s.options.LogDebug {
		log.Printf(
			"%s: added async variant handle=%d, index=%d, count=%d, group=%+v",
			s.options.LogPrefix,
			handle,
			index,
			v.SelectCount,
			v.GroupSlice,
		)
	}
}

func (s *Scheduler[G]) removeAsyncVariant(v *AsyncVariant[G]) func() {
	handle := v.asyncHandle
	index := v.selectIndex

	if v.inRemove {
		log.Printf(
			"%s: handle=%d, index=%d, count=%d, group=%+v, already removed",
			s.options.LogPrefix,
			handle,
			index,
			v.SelectCount,
			v.GroupSlice,
		)
		return func() {}
	}
	v.inRemove = true

	f := func() {
		if v.releaseFunctor == nil {
			return
		}

		if s.options.LogDebug {
			log.Printf(
				"%s: invoking release handle=%d, index=%d, count=%d, group=%+v",
				s.options.LogPrefix,
				handle,
				index,
				v.SelectCount,
				v.GroupSlice,
			)
		}

		func() {
			defer func() {
				rec := recover()
				if rec != nil {
					log.Printf(
						"%s: handle=%d, index=%d, count=%d, group=%+v, release functor recovered from panic: %+v",
						s.options.LogPrefix,
						handle,
						index,
						v.SelectCount,
						v.GroupSlice,
						rec,
					)
				}
			}()
			v.releaseFunctor(s, v)
		}()
	}

	for _, group := range v.GroupSlice {
		context, found := s.groupContextMap[group]
		if !found {
			log.Printf("%s: group=%+v not found in groupContextMap", s.options.LogPrefix, group)
			continue
		}

		context.asyncHandleTree.Remove(handle)

		if context.asyncHandleTree.Empty() {
			delete(s.groupContextMap, group)
		}
	}

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
			log.Printf("%s: no async variant found for swapIndex=%d", s.options.LogPrefix, swapIndex)
			return f
		}
		if swapVariant.selectIndex != swapIndex {
			log.Printf("%s: swapVariant selectIndex=%d mismatch swapIndex=%d", s.options.LogPrefix, swapVariant.selectIndex, swapIndex)
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
			"%s: removed async variant handle=%d, index=%d, count=%d, group=%+v",
			s.options.LogPrefix,
			handle,
			index,
			v.SelectCount,
			v.GroupSlice,
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
		log.Printf("%s: release all async variants, size=%d", s.options.LogPrefix, s.selectIndexTree.Size())
	}

	scopedIndexTree := rbt.New[uint16, *AsyncVariant[G]]()

	it := s.selectIndexTree.Iterator()
	for it.Next() {
		scopedIndexTree.Put(it.Key(), it.Value())
	}

	s.releaseAsyncVariantByTree(scopedIndexTree)
}

func (s *Scheduler[G]) releaseGroup(group G) {
	context, found := s.groupContextMap[group]
	if !found {
		if s.options.LogDebug {
			log.Printf("%s: no async variant found for group=%+v", s.options.LogPrefix, group)
		}
		return
	}

	if s.options.LogDebug {
		log.Printf("%s: release group=%+v async variants, size=%d", s.options.LogPrefix, group, context.asyncHandleTree.Size())
	}

	scopedIndexTree := rbt.New[uint16, *AsyncVariant[G]]()

	it := context.asyncHandleTree.Iterator()
	for it.Next() {
		v := it.Value()
		scopedIndexTree.Put(v.selectIndex, v)
	}

	s.releaseAsyncVariantByTree(scopedIndexTree)
}

func (s *Scheduler[G]) releaseGroupSlice(groupSlice []G) {
	scopedIndexTree := rbt.New[uint16, *AsyncVariant[G]]()

	for _, group := range groupSlice {
		func() {
			context, found := s.groupContextMap[group]
			if !found {
				return
			}

			it := context.asyncHandleTree.Iterator()
			for it.Next() {
				v := it.Value()

				// note that one async variant may associate with multiple groups, here we collect unique occurrences
				scopedIndexTree.Put(v.selectIndex, v)
			}
		}()
	}

	if s.options.LogDebug {
		log.Printf("%s: release group=%+v async variants, size=%d", s.options.LogPrefix, groupSlice, scopedIndexTree.Size())
	}

	s.releaseAsyncVariantByTree(scopedIndexTree)
}

func (s *Scheduler[G]) scheduleAsyncEvent(event *ScheduleAsyncEvent[G]) {
	s.addAsyncVariant(event.AsyncVariant)
}

func (s *Scheduler[G]) scheduleSequenceEvent(event *ScheduleSequenceEvent[G]) {
	event.Sequence.enter(nil)
}

// must be invoked on same goroutine as processLoop
func (s *Scheduler[G]) ProcessSync(event Event) {
	s.handle(event)
}

// can be invoked on any goroutine
func (s *Scheduler[G]) ProcessAsync(event Event) {
	select {
	case s.eventch <- event:
	default:
		log.Printf("%s: failed to push to eventch", s.options.LogPrefix)
	}
}
