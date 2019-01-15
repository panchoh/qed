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
	"bytes"
	"runtime/debug"
	"time"

	"github.com/bbva/qed/balloon/navigator"
	"github.com/bbva/qed/hashing"
	"github.com/bbva/qed/storage"
	"github.com/coocood/freecache"
	metrics "github.com/rcrowley/go-metrics"
)

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

func (c FreeCache) Get(pos navigator.Position) (hashing.Digest, bool) {
	ts := time.Now()
	value, err := c.cached.Get(pos.Bytes())
	if err != nil {
		c.gets.UpdateSince(ts)
		return nil, false
	}
	c.gets.UpdateSince(ts)
	return value, true
}

func (c *FreeCache) Put(pos navigator.Position, value hashing.Digest) {
	ts := time.Now()
	c.cached.Set(pos.Bytes(), value, 0)
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
