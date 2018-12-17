package hyper2

import (
	"bytes"
	"runtime/debug"
	"time"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/allegro/bigcache"
	"github.com/bbva/qed/hashing"
	"github.com/bbva/qed/log"
	"github.com/bbva/qed/storage"
	"github.com/coocood/freecache"
	metrics "github.com/rcrowley/go-metrics"
)

type Cache interface {
	Get(key []byte) (hashing.Digest, bool)
}

type ModifiableCache interface {
	Put(key []byte, value hashing.Digest)
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

func (c FastCache) Get(key []byte) (hashing.Digest, bool) {
	ts := time.Now()
	value := c.cached.Get(nil, key)
	if value == nil {
		c.gets.UpdateSince(ts)
		return nil, false
	}
	c.gets.UpdateSince(ts)
	return value, true
}

func (c *FastCache) Put(key []byte, value hashing.Digest) {
	ts := time.Now()
	c.cached.Set(key, value)
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

type FreeCache struct {
	cached *freecache.Cache

	gets metrics.Timer
	puts metrics.Timer
}

func NewFreeCache(initialSize int) *FreeCache {
	cache := freecache.NewCache(initialSize)
	debug.SetGCPercent(20)
	gets := metrics.NewTimer()
	puts := metrics.NewTimer()
	metrics.Register("cache.gets", gets)
	metrics.Register("cache.puts", puts)
	return &FreeCache{cached: cache, gets: gets, puts: puts}
}

func (c FreeCache) Get(key []byte) (hashing.Digest, bool) {
	ts := time.Now()
	value, err := c.cached.Get(key)
	if err != nil {
		c.gets.UpdateSince(ts)
		return nil, false
	}
	c.gets.UpdateSince(ts)
	return value, true
}

func (c *FreeCache) Put(key []byte, value hashing.Digest) {
	ts := time.Now()
	c.cached.Set(key, value, 0)
	c.puts.UpdateSince(ts)
}

func (c *FreeCache) Fill(r storage.KVPairReader) (err error) {
	defer r.Close()
	for {
		entries := make([]*storage.KVPair, 100)
		n, err := r.Read(entries)
		if err != nil || n == 0 {
			break
		}
		for _, entry := range entries {
			if entry != nil {
				c.cached.Set(entry.Key, entry.Value, 0)
			}
		}
	}
	return nil
}

func (c FreeCache) Size() int {
	return int(c.cached.EntryCount())
}

func (c FreeCache) Equal(o *FreeCache) bool {
	it := c.cached.NewIterator()
	entry := it.Next()
	for entry != nil {
		v2, err := o.cached.Get(entry.Key)
		if err != nil {
			return false
		}
		if !bytes.Equal(entry.Value, v2) {
			return false
		}
		entry = it.Next()
	}
	return true
}

type BigCache struct {
	cached *bigcache.BigCache

	gets metrics.Timer
	puts metrics.Timer
}

func NewBigCache(maxEntries, maxEntrySize int64) *BigCache {
	config := bigcache.DefaultConfig(10 * time.Minute)
	config.MaxEntriesInWindow = int(maxEntries)
	config.MaxEntrySize = int(maxEntrySize)
	config.HardMaxCacheSize = int(maxEntries) * 100
	config.CleanWindow = -1
	config.Logger = log.GetLogger()
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

func (c BigCache) Get(key []byte) (hashing.Digest, bool) {
	ts := time.Now()
	value, err := c.cached.Get(string(key))
	if err != nil {
		c.gets.UpdateSince(ts)
		return nil, false
	}
	c.gets.UpdateSince(ts)
	return value, true
}

func (c *BigCache) Put(key []byte, value hashing.Digest) {
	ts := time.Now()
	c.cached.Set(string(key), value)
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

const keySize = 34
const valueSize = 32

type FixedSizeCache struct {
	cached map[[keySize]byte][valueSize]byte

	gets metrics.Timer
	puts metrics.Timer
}

func NewFixedSizeCache(numEntries uint64) *FixedSizeCache {
	gets := metrics.NewTimer()
	puts := metrics.NewTimer()
	metrics.Register("cache.gets", gets)
	metrics.Register("cache.puts", puts)
	return &FixedSizeCache{
		cached: make(map[[keySize]byte][valueSize]byte, numEntries),
		gets:   gets,
		puts:   puts,
	}
}

func (c FixedSizeCache) Get(key []byte) (hashing.Digest, bool) {
	ts := time.Now()
	var k [keySize]byte
	copy(k[:], key[:keySize])
	digest, ok := c.cached[k]
	if !ok {
		c.gets.UpdateSince(ts)
		return nil, false
	}
	c.gets.UpdateSince(ts)
	return digest[:valueSize], true
}

func (c *FixedSizeCache) Put(key []byte, value hashing.Digest) {
	ts := time.Now()
	var k [keySize]byte
	var v [valueSize]byte
	copy(k[:], key[:keySize])
	copy(v[:], value[:valueSize])
	c.cached[k] = v
	c.puts.UpdateSince(ts)
}

func (c *FixedSizeCache) Fill(r storage.KVPairReader) (err error) {
	defer r.Close()
	for {
		entries := make([]*storage.KVPair, 100)
		n, err := r.Read(entries)
		if err != nil || n == 0 {
			break
		}
		for _, entry := range entries {
			if entry != nil {
				var key [keySize]byte
				var value [valueSize]byte
				copy(key[:], entry.Key[:keySize])
				copy(value[:], entry.Value[:valueSize])
				c.cached[key] = value
			}
		}
	}
	return nil
}

func (c FixedSizeCache) Size() int {
	return len(c.cached)
}

func (c FixedSizeCache) Equal(o *FixedSizeCache) bool {
	for k, v1 := range c.cached {
		v2, ok := o.cached[k]
		if !ok {
			return false
		}
		if !bytes.Equal(v1[:valueSize], v2[:valueSize]) {
			return false
		}
	}
	return true
}
