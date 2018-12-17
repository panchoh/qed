package cache

import (
	"bytes"
	"log"
	"time"

	"github.com/allegro/bigcache"
	"github.com/bbva/qed/balloon/navigator"
	"github.com/bbva/qed/hashing"
	"github.com/bbva/qed/storage"
	metrics "github.com/rcrowley/go-metrics"
)

type BigCache struct {
	cached *bigcache.BigCache

	gets metrics.Timer
	puts metrics.Timer
}

func NewBigCache(maxEntries, maxEntrySize int64) *BigCache {
	config := bigcache.DefaultConfig(10 * time.Minute)
	config.MaxEntriesInWindow = int(maxEntrySize)
	config.MaxEntrySize = int(maxEntries)
	config.HardMaxCacheSize = int(maxEntries) * 100
	cache, err := bigcache.NewBigCache(config)
	if err != nil {
		log.Fatalf("%+v", err)
	}
	gets := metrics.NewTimer()
	puts := metrics.NewTimer()
	metrics.Register("cache.gets", gets)
	metrics.Register("cache.puts", puts)
	return &BigCache{cached: cache, gets: gets, puts: puts}
}

func (c BigCache) Get(pos navigator.Position) (hashing.Digest, bool) {
	ts := time.Now()
	value, err := c.cached.Get(pos.StringId())
	if err != nil {
		return nil, false
	}
	c.gets.UpdateSince(ts)
	return value, true
}

func (c *BigCache) Put(pos navigator.Position, value hashing.Digest) {
	ts := time.Now()
	c.cached.Set(pos.StringId(), value)
	c.puts.UpdateSince(ts)
}

func (c *BigCache) Fill(r storage.KVPairReader) (err error) {
	defer r.Close()
	for {
		entries := make([]*storage.KVPair, 100)
		n, err := r.Read(entries)
		if err != nil || n == 0 {
			break
		}
		for _, entry := range entries {
			if entry != nil {
				c.cached.Set(string(entry.Key), entry.Value)
			}
		}
	}
	return nil
}

func (c BigCache) Size() int {
	return c.cached.Len()
}

func (c BigCache) Equal(o *BigCache) bool {
	it := c.cached.Iterator()
	for it.SetNext() {
		entry, _ := it.Value()
		v2, err := o.cached.Get(entry.Key())
		if err != nil {
			return false
		}
		if !bytes.Equal(entry.Value(), v2) {
			return false
		}
	}
	return true
}
