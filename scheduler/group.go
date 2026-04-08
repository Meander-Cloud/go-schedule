package scheduler

import (
	"fmt"
	"strings"

	rbt "github.com/emirpasic/gods/v2/trees/redblacktree"
)

type GroupTracker[G comparable] struct {
	// level oriented group tracking, with holder groups at base level zero
	levelGroupTree *rbt.Tree[uint16, *[]G]

	// unique groups seen across all levels
	uniqueGroupMap map[G]struct{}

	// lazily populated as needed
	printCache string
}

type GroupContext[G comparable] struct {
	group G

	// handle tree of outstanding async variants that belong to this group
	asyncHandleTree *rbt.Tree[uint16, *AsyncVariant[G]]
}

func (p *GroupTracker[G]) initGroupSlice(groupSlicePtr *[]G) {
	if len(*groupSlicePtr) == 0 {
		return
	}

	p.levelGroupTree.Put(
		0,
		groupSlicePtr,
	)

	for _, group := range *groupSlicePtr {
		p.uniqueGroupMap[group] = struct{}{}
	}
}

func NewGroupTracker[G comparable](groupSlice []G) *GroupTracker[G] {
	p := &GroupTracker[G]{
		levelGroupTree: rbt.New[uint16, *[]G](),
		uniqueGroupMap: make(map[G]struct{}),
		printCache:     "",
	}

	p.initGroupSlice(&groupSlice)

	return p
}

func newGroupTrackerWithParent[G comparable](parent *GroupTracker[G], groupSlice []G) *GroupTracker[G] {
	p := &GroupTracker[G]{
		levelGroupTree: rbt.New[uint16, *[]G](),
		uniqueGroupMap: make(map[G]struct{}),
		printCache:     "",
	}

	func() {
		if parent == nil {
			return
		}

		it := parent.levelGroupTree.Iterator()
		for it.Next() {
			p.levelGroupTree.Put(
				it.Key()+1, // increment levels to correctly reflect ancestry
				it.Value(),
			)
		}

		for group := range parent.uniqueGroupMap {
			p.uniqueGroupMap[group] = struct{}{}
		}
	}()

	p.initGroupSlice(&groupSlice)

	return p
}

func (p *GroupTracker[G]) printLevel(sb *strings.Builder, level uint16, groupSlicePtr *[]G) {
	if level > 0 {
		fmt.Fprintf(
			sb,
			"<+%d>[",
			level,
		)
	} else {
		sb.WriteByte('[')
	}

	first := true
	for _, group := range *groupSlicePtr {
		if first {
			first = false
		} else {
			sb.WriteByte(',')
		}

		fmt.Fprintf(
			sb,
			"%+v",
			group,
		)
	}

	sb.WriteByte(']')
}

func (p *GroupTracker[G]) printOneLevel() string {
	sb := new(strings.Builder)
	sb.Grow(256)

	node := p.levelGroupTree.Left()

	p.printLevel(
		sb,
		node.Key,
		node.Value,
	)

	p.printCache = sb.String()
	return p.printCache
}

func (p *GroupTracker[G]) printMultiLevel() string {
	sb := new(strings.Builder)
	sb.Grow(256)

	it := p.levelGroupTree.Iterator()
	it.End()

	sb.WriteByte('[')

	first := true
	for it.Prev() {
		if first {
			first = false
		} else {
			sb.WriteByte(',')
		}

		p.printLevel(
			sb,
			it.Key(),
			it.Value(),
		)
	}

	sb.WriteByte(']')

	p.printCache = sb.String()
	return p.printCache
}

func (p *GroupTracker[G]) Print() string {
	if p == nil {
		return "nil"
	}

	if p.printCache != "" {
		return p.printCache
	}

	switch p.levelGroupTree.Size() {
	case 0:
		p.printCache = "[]"
		return p.printCache
	case 1:
		return p.printOneLevel()
	default:
		return p.printMultiLevel()
	}
}
