package hyper2

import (
	"sync"
	"time"

	"github.com/bbva/qed/hashing"
	"github.com/bbva/qed/log"
	"github.com/bbva/qed/storage"
	"github.com/bbva/qed/util"
	"github.com/rcrowley/go-metrics"
)

const (
	CacheSize int64 = (1 << 26) * 70 //68 // 2^26 elements * 68 bytes each entry
)

type HyperTree struct {
	store         storage.Store
	cache         ModifiableCache
	hasherF       func() hashing.Hasher
	cacheLevel    uint16
	defaultHashes []hashing.Digest
	hasher        hashing.Hasher

	sync.RWMutex

	addTime      metrics.Timer
	pruningTime  metrics.Timer
	visitingTime metrics.Timer
	pruningStats *PruningStats
}

func NewHyperTree(hasherF func() hashing.Hasher, store storage.Store, cache ModifiableCache) *HyperTree {
	hasher := hasherF()
	//cacheLevel := hasher.Len() - uint16(math.Max(float64(2), math.Floor(float64(hasher.Len())/10)))
	cacheLevel := hasher.Len() - 25
	tree := &HyperTree{
		store:         store,
		cache:         cache,
		hasherF:       hasherF,
		cacheLevel:    cacheLevel,
		defaultHashes: make([]hashing.Digest, hasher.Len()),
		hasher:        hasher,
	}

	tree.defaultHashes[0] = tree.hasher.Do([]byte{0x0}, []byte{0x0})
	for i := uint16(1); i < hasher.Len(); i++ {
		tree.defaultHashes[i] = tree.hasher.Do(tree.defaultHashes[i-1], tree.defaultHashes[i-1])
	}

	// warm-up cache
	tree.RebuildCache()

	tree.addTime = metrics.NewTimer()
	tree.pruningTime = metrics.NewTimer()
	tree.visitingTime = metrics.NewTimer()
	metrics.Register("hyper.add", tree.addTime)
	metrics.Register("hyper.pruning", tree.pruningTime)
	metrics.Register("hyper.visiting", tree.visitingTime)

	tree.pruningStats = &PruningStats{
		ThroughCache: metrics.NewTimer(),
		AfterCache:   metrics.NewTimer(),
		GetLeaves:    metrics.NewTimer(),
		Leaves:       metrics.NewHistogram(metrics.NewExpDecaySample(1028, 0.015)),
	}
	metrics.Register("hyper.pruning.though_cache", tree.pruningStats.ThroughCache)
	metrics.Register("hyper.pruning.after_cache", tree.pruningStats.AfterCache)
	metrics.Register("hyper.pruning.get_leaves", tree.pruningStats.GetLeaves)
	metrics.Register("hyper.pruning.leaves", tree.pruningStats.Leaves)

	return tree
}

func (t *HyperTree) Add(eventDigest hashing.Digest, version uint64) (hashing.Digest, []*storage.Mutation, error) {

	ts1 := time.Now()

	t.Lock()
	defer t.Unlock()

	// visitors
	computeHash := NewComputeHashVisitor(t.hasher)
	caching := NewCachingVisitor(computeHash, t.cache)
	collect := NewCollectMutationsVisitor(caching)

	// build pruning context
	versionAsBytes := util.Uint64AsBytes(version)
	context := PruningContext{
		navigator:     NewHyperTreeNavigator(t.hasher.Len()),
		cacheResolver: NewSingleTargetedCacheResolver(t.hasher.Len(), t.cacheLevel, eventDigest),
		cache:         t.cache,
		store:         t.store,
		defaultHashes: t.defaultHashes,
		stats:         t.pruningStats,
	}

	ts2 := time.Now()
	result, err := NewInsertPruner(eventDigest, versionAsBytes, context).PruneAndVisit(collect)
	if err != nil {
		return nil, nil, err
	}
	rootHash := result.(hashing.Digest)
	t.visitingTime.Update(time.Since(ts2))

	// create a mutation for the new leaf
	//leafMutation := storage.NewMutation(storage.IndexPrefix, eventDigest, versionAsBytes)

	// collect mutations
	//mutations := append(collect.Result(), leafMutation)

	t.addTime.Update(time.Since(ts1))

	return rootHash, collect.Result(), nil
}

func (t *HyperTree) RebuildCache() error {
	t.Lock()
	defer t.Unlock()

	// warm up cache
	log.Info("Warming up hyper cache...")

	// Fill last cache level with stored data
	err := t.cache.Fill(t.store.GetAll(storage.HyperCachePrefix))
	if err != nil {
		return err
	}

	if t.cache.Size() == 0 { // nothing to recompute
		log.Infof("Warming up done, elements cached: %d", t.cache.Size())
		return nil
	}

	// Recompute and fill the rest of the cache
	navigator := NewHyperTreeNavigator(t.hasher.Len())
	root := navigator.Root()
	// skip root
	t.populateCache(navigator.GoToLeft(root), navigator)
	t.populateCache(navigator.GoToRight(root), navigator)
	log.Infof("Warming up done, elements cached: %d", t.cache.Size())
	return nil
}

func (t *HyperTree) populateCache(pos *Position, nav *HyperTreeNavigator) hashing.Digest {
	if pos.Height == t.cacheLevel {
		cached, ok := t.cache.Get(pos.Bytes())
		if !ok {
			return nil
		}
		return cached
	}
	leftPos := nav.GoToLeft(pos)
	rightPos := nav.GoToRight(pos)
	left := t.populateCache(leftPos, nav)
	right := t.populateCache(rightPos, nav)

	if left == nil && right == nil {
		return nil
	}
	if left == nil {
		left = t.defaultHashes[leftPos.Height]
	}
	if right == nil {
		right = t.defaultHashes[rightPos.Height]
	}

	digest := t.hasher.Salted(pos.Bytes(), left, right)
	t.cache.Put(pos.Bytes(), digest)
	return digest
}

func (t *HyperTree) Add2(eventDigest hashing.Digest, version uint64) (hashing.Digest, error) {

	ts1 := time.Now()

	t.Lock()
	defer t.Unlock()

	// visitors
	computeHash := NewComputeHashVisitor(t.hasher)
	caching := NewCachingVisitor(computeHash, t.cache)

	// build pruning context
	versionAsBytes := util.Uint64AsBytes(version)
	context := PruningContext{
		navigator:     NewHyperTreeNavigator(t.hasher.Len()),
		cacheResolver: NewSingleTargetedCacheResolver(t.hasher.Len(), t.cacheLevel, eventDigest),
		cache:         t.cache,
		store:         t.store,
		defaultHashes: t.defaultHashes,
		stats:         nil,
	}

	ts2 := time.Now()
	result, err := NewInsertPruner2(eventDigest, versionAsBytes, context).PruneAndVisit(caching)
	if err != nil {
		return nil, err
	}
	rootHash := result.(hashing.Digest)
	t.visitingTime.Update(time.Since(ts2))
	t.addTime.Update(time.Since(ts1))

	return rootHash, nil
}
