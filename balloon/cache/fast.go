/*
   Copyright 2018 Banco Bilbao Vizcaya Argentaria, S.A.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package cache

import (
	"time"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/bbva/qed/balloon/navigator"
	"github.com/bbva/qed/hashing"
	"github.com/bbva/qed/storage"
	"github.com/rcrowley/go-metrics"
)

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

func (c FastCache) Get(pos navigator.Position) (hashing.Digest, bool) {
	ts := time.Now()
	value := c.cached.Get(nil, pos.Bytes())
	if value == nil {
		c.gets.UpdateSince(ts)
		return nil, false
	}
	c.gets.UpdateSince(ts)
	return value, true
}

func (c *FastCache) Put(pos navigator.Position, value hashing.Digest) {
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
