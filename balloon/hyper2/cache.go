package hyper2

import (
	"time"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/bbva/qed/hashing"
	"github.com/bbva/qed/storage"
	metrics "github.com/rcrowley/go-metrics"
)

type Cache interface {
	Get(pos *Position) (hashing.Digest, bool)
}

type ModifiableCache interface {
	Put(pos *Position, value hashing.Digest)
	Fill(r storage.KVPairReader) error
	Size() int
	Cache
}

type FastCache struct {
	cached *fastcache.Cache

	gets metrics.Timer
	puts metrics.Timer
}

func NewFastCache(maxBytes int64) *FastCache {
	cache := fastcache.New(int(maxBytes))
	gets := metrics.NewTimer()
	puts := metrics.NewTimer()
	metrics.Register("cache.gets", gets)
	metrics.Register("cache.puts", puts)
	return &FastCache{cached: cache, gets: gets, puts: puts}
}

func (c FastCache) Get(pos *Position) (hashing.Digest, bool) {
	ts := time.Now()
	value := c.cached.Get(nil, pos.Bytes())
	if value == nil {
		c.gets.UpdateSince(ts)
		return nil, false
	}
	c.gets.UpdateSince(ts)
	return value, true
}

func (c *FastCache) Put(pos *Position, value hashing.Digest) {
	ts := time.Now()
	c.cached.Set(pos.Bytes(), value)
	c.puts.UpdateSince(ts)
}

func (c *FastCache) Fill(r storage.KVPairReader) (err error) {
	defer r.Close()
	for {
		entries := make([]*storage.KVPair, 100)
		n, err := r.Read(entries)
		if err != nil || n == 0 {
			break
		}
		for _, entry := range entries {
			if entry != nil {
				c.cached.Set(entry.Key, entry.Value)
			}
		}
	}
	return nil
}

func (c FastCache) Size() int {
	var s fastcache.Stats
	c.cached.UpdateStats(&s)
	return int(s.EntriesCount)
}

func (c FastCache) Equal(o *FastCache) bool {
	// can only check size and entries count
	var stats, oStats fastcache.Stats
	c.cached.UpdateStats(&stats)
	o.cached.UpdateStats(&oStats)
	return stats.BytesSize == oStats.BytesSize && stats.EntriesCount == oStats.EntriesCount
}
