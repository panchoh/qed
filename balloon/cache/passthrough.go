/*
   Copyright 2018-2019 Banco Bilbao Vizcaya Argentaria, S.A.

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
	"github.com/bbva/qed/storage"
)

type PassThroughCache struct {
	table storage.Table
	store storage.Store
}

func NewPassThroughCache(table storage.Table, store storage.Store) *PassThroughCache {
	return &PassThroughCache{
		table: table,
		store: store,
	}
}

func (c PassThroughCache) Get(key []byte) ([]byte, bool) {
	pair, err := c.store.Get(c.table, key)
	if err != nil {
		return nil, false
	}
	return pair.Value, true
}
