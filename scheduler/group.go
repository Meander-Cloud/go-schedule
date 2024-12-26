package scheduler

import (
	rbt "github.com/emirpasic/gods/v2/trees/redblacktree"
)

type GroupContext[G comparable] struct {
	group G

	// handle tree of outstanding async variants that belong to this group
	asyncHandleTree *rbt.Tree[uint16, *AsyncVariant[G]]
}
