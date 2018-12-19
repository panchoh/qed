package hyper2

import (
	"errors"
	"time"

	"github.com/bbva/qed/hashing"
	"github.com/bbva/qed/storage"
	"github.com/bbva/qed/util"
	"github.com/rcrowley/go-metrics"
)

var (
	ErrLeavesSlice = errors.New("this should never happen (unsorted LeavesSlice or broken split?)")
)

type PruningStats struct {
	ThroughCache metrics.Timer
	GetLeaves    metrics.Timer
	AfterCache   metrics.Timer
	Leaves       metrics.Histogram
}

type PruningContext struct {
	navigator     *HyperTreeNavigator
	cacheResolver CacheResolver
	cache         Cache
	store         storage.Store
	defaultHashes []hashing.Digest
	stats         *PruningStats
}

type InsertPruner struct {
	key   hashing.Digest
	value []byte
	PruningContext
}

func NewInsertPruner(key, value []byte, context PruningContext) *InsertPruner {
	return &InsertPruner{key, value, context}
}

func (p *InsertPruner) PruneAndVisit(v PostOrderVisitor) (interface{}, error) {
	leaves := storage.KVRange{storage.NewKVPair(p.key, p.value)}
	root := p.navigator.Root()
	left, err := p.traverseAndVisit(p.navigator.GoToLeft(root), leaves, v)
	if err != nil {
		return nil, err
	}
	right, err := p.traverseAndVisit(p.navigator.GoToRight(root), leaves, v)
	if err != nil {
		return nil, err
	}
	return v.VisitRoot(root, left, right), nil
}

func (p *InsertPruner) traverseAndVisit(pos *Position, leaves storage.KVRange, v PostOrderVisitor) (interface{}, error) {
	if p.cacheResolver.ShouldBeInCache(pos) {
		digest, ok := p.cache.Get(pos.Bytes())
		if !ok {
			return v.VisitCached(pos, p.defaultHashes[pos.Height]), nil
		}
		return v.VisitCached(pos, digest), nil
	}

	// if we are over the cache level, we need to do a range query to get the leaves
	var atLastLevel bool
	if atLastLevel = p.cacheResolver.ShouldCache(pos); atLastLevel {
		ts := time.Now()
		first := p.navigator.DescendToFirst(pos)
		last := p.navigator.DescendToLast(pos)

		kvRange, _ := p.store.GetRange(storage.IndexPrefix, first.Index, last.Index)
		p.stats.Leaves.Update(int64(len(kvRange)))

		// replace leaves with new slice and append the previous to the new one
		for _, l := range leaves {
			kvRange = kvRange.InsertSorted(l)
		}
		leaves = kvRange
		p.stats.GetLeaves.UpdateSince(ts)
	}

	rightPos := p.navigator.GoToRight(pos)
	leftPos := p.navigator.GoToLeft(pos)
	leftSlice, rightSlice := leaves.Split(rightPos.Index)

	var left, right interface{}
	var err error
	if atLastLevel {
		ts := time.Now()
		left, err = p.traverseWithoutCacheAndVisit(leftPos, leftSlice, v)
		if err != nil {
			return nil, err
		}
		right, err = p.traverseWithoutCacheAndVisit(rightPos, rightSlice, v)
		p.stats.AfterCache.UpdateSince(ts)
	} else {
		left, err = p.traverseAndVisit(leftPos, leftSlice, v)
		if err != nil {
			return nil, err
		}
		right, err = p.traverseAndVisit(rightPos, rightSlice, v)
	}
	if err != nil {
		return nil, err
	}

	result := v.VisitCacheable(pos, v.VisitNode(pos, left, right))
	if p.cacheResolver.ShouldCollect(pos) {
		return v.VisitMutable(pos, result, storage.HyperCachePrefix, storage.Set), nil
	}

	return result, nil
}

func (p *InsertPruner) traverseWithoutCacheAndVisit(pos *Position, leaves storage.KVRange, v PostOrderVisitor) (interface{}, error) {

	numLeaves := len(leaves)
	switch {
	case numLeaves == 0:
		return v.VisitCached(pos, p.defaultHashes[pos.Height]), nil
	case numLeaves == 1:
		if p.cacheResolver.IsOnPath(pos) { // if its on the path of insertion then it's a new leaf
			// Store the leaf at the higher level
			return v.VisitMutable(pos, v.VisitLeaf(pos, leaves[0].Value), storage.IndexPrefix, storage.Set), nil
		}
		// it's a previously inserted leaf so we have to check its original height
		// to figure out if we have to push down the leaf to a lower level
		k := leaves[0].Key
		previousPos := NewPosition(k[:len(k)-2], util.BytesAsUint16(k[len(k)-2:]))
		if previousPos.Height > pos.Height {
			// push down
			v.VisitMutable(previousPos, v.VisitLeaf(pos, leaves[0].Value), storage.IndexPrefix, storage.Delete)
			return v.VisitMutable(pos, v.VisitLeaf(pos, leaves[0].Value), storage.IndexPrefix, storage.Set), nil
		}
		return v.VisitLeaf(pos, leaves[0].Value), nil
	case numLeaves > 1:
		fallthrough
	default:
		if p.navigator.IsLeaf(pos) {
			return nil, ErrLeavesSlice
		}
		// descend to children
		rightPos := p.navigator.GoToRight(pos)
		leftSlice, rightSlice := leaves.Split(rightPos.Index)
		left, err := p.traverseWithoutCacheAndVisit(p.navigator.GoToLeft(pos), leftSlice, v)
		if err != nil {
			return nil, ErrLeavesSlice
		}
		right, err := p.traverseWithoutCacheAndVisit(rightPos, rightSlice, v)
		if err != nil {
			return nil, ErrLeavesSlice
		}

		return v.VisitNode(pos, left, right), nil
	}
}

type InsertPruner2 struct {
	key   hashing.Digest
	value []byte
	PruningContext
}

func NewInsertPruner2(key, value []byte, context PruningContext) *InsertPruner2 {
	return &InsertPruner2{key, value, context}
}

func (p *InsertPruner2) PruneAndVisit(v PostOrderVisitor) (interface{}, error) {
	return p.traverseAndVisit(p.navigator.Root(), v)
}

func (p *InsertPruner2) traverseAndVisit(pos *Position, v PostOrderVisitor) (interface{}, error) {

	if p.cacheResolver.ShouldBeInCache(pos) {
		digest, ok := p.cache.Get(pos.Bytes())
		if !ok {
			return v.VisitCached(pos, p.defaultHashes[pos.Height]), nil
		}
		return v.VisitCached(pos, digest), nil
	}

	// if we are over the cache level, we need to do a range query to get the leaves
	var atLastLevel bool
	if atLastLevel = p.cacheResolver.ShouldCache(pos); atLastLevel {
		return v.VisitCacheable(pos, v.VisitLeaf(pos, p.value)), nil
	}

	rightPos := p.navigator.GoToRight(pos)
	leftPos := p.navigator.GoToLeft(pos)

	var left, right interface{}
	var err error

	left, err = p.traverseAndVisit(leftPos, v)
	if err != nil {
		return nil, err
	}
	right, err = p.traverseAndVisit(rightPos, v)
	if err != nil {
		return nil, err
	}

	if p.navigator.IsRoot(pos) {
		return v.VisitRoot(pos, left, right), nil
	}

	result := v.VisitCacheable(pos, v.VisitNode(pos, left, right))

	return result, nil
}
