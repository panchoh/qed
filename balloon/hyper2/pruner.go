package hyper2

import (
	"errors"

	"github.com/bbva/qed/hashing"
	"github.com/bbva/qed/storage"
)

var (
	ErrLeavesSlice = errors.New("this should never happen (unsorted LeavesSlice or broken split?)")
)

type PruningContext struct {
	navigator     *HyperTreeNavigator
	cacheResolver CacheResolver
	cache         Cache
	store         storage.Store
	defaultHashes []hashing.Digest
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
	return p.traverseAndVisit(p.navigator.Root(), leaves, v)
}

func (p *InsertPruner) traverseAndVisit(pos *Position, leaves storage.KVRange, v PostOrderVisitor) (interface{}, error) {

	if p.cacheResolver.ShouldBeInCache(pos) {
		digest, ok := p.cache.Get(pos)
		if !ok {
			return v.VisitCached(pos, p.defaultHashes[pos.Height()]), nil
		}
		return v.VisitCached(pos, digest), nil
	}

	// if we are over the cache level, we need to do a range query to get the leaves
	var atLastLevel bool
	if atLastLevel = p.cacheResolver.ShouldCache(pos); atLastLevel {
		first := p.navigator.DescendToFirst(pos)
		last := p.navigator.DescendToLast(pos)

		kvRange, _ := p.store.GetRange(storage.IndexPrefix, first.Index(), last.Index())

		// replace leaves with new slice and append the previous to the new one
		for _, l := range leaves {
			kvRange = kvRange.InsertSorted(l)
		}
		leaves = kvRange
	}

	rightPos := p.navigator.GoToRight(pos)
	leftPos := p.navigator.GoToLeft(pos)
	leftSlice, rightSlice := leaves.Split(rightPos.Index())

	var left, right interface{}
	var err error
	if atLastLevel {
		left, err = p.traverseWithoutCacheAndVisit(leftPos, leftSlice, v)
		if err != nil {
			return nil, err
		}
		right, err = p.traverseWithoutCacheAndVisit(rightPos, rightSlice, v)
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

	if p.navigator.IsRoot(pos) {
		return v.VisitRoot(pos, left, right), nil
	}

	result := v.VisitCacheable(pos, v.VisitNode(pos, left, right))
	if p.cacheResolver.ShouldCollect(pos) {
		return v.VisitCollectable(pos, result), nil
	}

	return result, nil
}

func (p *InsertPruner) traverseWithoutCacheAndVisit(pos *Position, leaves storage.KVRange, v PostOrderVisitor) (interface{}, error) {
	if p.navigator.IsLeaf(pos) && len(leaves) == 1 {
		return v.VisitLeaf(pos, leaves[0].Value), nil
	}
	if !p.navigator.IsRoot(pos) && len(leaves) == 0 {
		return v.VisitCached(pos, p.defaultHashes[pos.Height()]), nil
	}
	if len(leaves) > 1 && p.navigator.IsLeaf(pos) {
		return nil, ErrLeavesSlice
	}

	// we do a post-order traversal

	// split leaves
	rightPos := p.navigator.GoToRight(pos)
	leftSlice, rightSlice := leaves.Split(rightPos.Index())
	left, err := p.traverseWithoutCacheAndVisit(p.navigator.GoToLeft(pos), leftSlice, v)
	if err != nil {
		return nil, ErrLeavesSlice
	}
	right, err := p.traverseWithoutCacheAndVisit(rightPos, rightSlice, v)
	if err != nil {
		return nil, ErrLeavesSlice
	}

	if p.navigator.IsRoot(pos) {
		return v.VisitRoot(pos, left, right), nil
	}
	return v.VisitNode(pos, left, right), nil
}
