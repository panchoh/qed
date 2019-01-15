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

package badger

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bbva/qed/storage"
	"github.com/bbva/qed/testutils/rand"
	"github.com/bbva/qed/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rcrowley/go-metrics"
)

func TestMutate(t *testing.T) {
	store, closeF := openBadgerStore(t)
	defer closeF()
	prefix := byte(0x0)

	tests := []struct {
		testname      string
		key, value    []byte
		expectedError error
	}{
		{"Mutate Key=Value", []byte("Key"), []byte("Value"), nil},
	}

	for _, test := range tests {
		err := store.Mutate([]*storage.Mutation{
			{prefix, test.key, test.value},
		})
		require.Equalf(t, test.expectedError, err, "Error mutating in test: %s", test.testname)
		_, err = store.Get(prefix, test.key)
		require.Equalf(t, test.expectedError, err, "Error getting key in test: %s", test.testname)
	}
}

func TestGetExistentKey(t *testing.T) {

	store, closeF := openBadgerStore(t)
	defer closeF()

	testCases := []struct {
		prefix        byte
		key, value    []byte
		expectedError error
	}{
		{byte(0x0), []byte("Key1"), []byte("Value1"), nil},
		{byte(0x0), []byte("Key2"), []byte("Value2"), nil},
		{byte(0x1), []byte("Key3"), []byte("Value3"), nil},
		{byte(0x1), []byte("Key4"), []byte("Value4"), storage.ErrKeyNotFound},
	}

	for _, test := range testCases {
		if test.expectedError == nil {
			err := store.Mutate([]*storage.Mutation{
				{test.prefix, test.key, test.value},
			})
			require.NoError(t, err)
		}

		stored, err := store.Get(test.prefix, test.key)
		if test.expectedError == nil {
			require.NoError(t, err)
			require.Equalf(t, stored.Key, test.key, "The stored key does not match the original: expected %d, actual %d", test.key, stored.Key)
			require.Equalf(t, stored.Value, test.value, "The stored value does not match the original: expected %d, actual %d", test.value, stored.Value)
		} else {
			require.Error(t, test.expectedError)
		}

	}

}

func TestGetRange(t *testing.T) {
	store, closeF := openBadgerStore(t)
	defer closeF()

	var testCases = []struct {
		size       int
		start, end byte
	}{
		{40, 10, 50},
		{0, 1, 9},
		{11, 1, 20},
		{10, 40, 60},
		{0, 60, 100},
		{0, 20, 10},
	}

	prefix := byte(0x0)
	for i := 10; i < 50; i++ {
		store.Mutate([]*storage.Mutation{
			{prefix, []byte{byte(i)}, []byte("Value")},
		})
	}

	for _, test := range testCases {
		slice, err := store.GetRange(prefix, []byte{test.start}, []byte{test.end})
		require.NoError(t, err)
		require.Equalf(t, len(slice), test.size, "Slice length invalid: expected %d, actual %d", test.size, len(slice))
	}

}

func TestDelete(t *testing.T) {
	store, closeF := openBadgerStore(t)
	defer closeF()

	prefix := byte(0x0)
	tests := []struct {
		testname      string
		key, value    []byte
		expectedError error
	}{
		{"Delete key", []byte("Key"), []byte("Value"), storage.ErrKeyNotFound},
	}

	for _, test := range tests {

		// err := store.Mutate({prefix, test.key, test.value))
		err := store.Mutate([]*storage.Mutation{
			{prefix, test.key, test.value},
		})
		require.NoError(t, err, "Error mutating in test: %s", test.testname)

		_, err = store.Get(prefix, test.key)
		require.NoError(t, err, "Error getting key in test: %s", test.testname)

		err = store.Delete(prefix, test.key)
		require.NoError(t, err, "Error deleting in test: %s", test.testname)

		_, err = store.Get(prefix, test.key)
		require.Equalf(t, test.expectedError, err, "Error getting non-existent key in test: %s", test.testname)
	}

}

func TestGetAll(t *testing.T) {

	prefix := storage.HyperCachePrefix
	numElems := uint16(1000)
	testCases := []struct {
		batchSize    int
		numBatches   int
		lastBatchLen int
	}{
		{10, 100, 10},
		{20, 50, 20},
		{17, 59, 14},
	}

	store, closeF := openBadgerStore(t)
	defer closeF()

	// insert
	for i := uint16(0); i < numElems; i++ {
		key := util.Uint16AsBytes(i)
		store.Mutate([]*storage.Mutation{
			{prefix, key, key},
		})
	}

	for i, c := range testCases {
		reader := store.GetAll(storage.HyperCachePrefix)
		numBatches := 0
		var lastBatchLen int
		for {
			entries := make([]*storage.KVPair, c.batchSize)
			n, _ := reader.Read(entries)
			if n == 0 {
				break
			}
			numBatches++
			lastBatchLen = n
		}
		reader.Close()
		assert.Equalf(t, c.numBatches, numBatches, "The number of batches should match for test case %d", i)
		assert.Equal(t, c.lastBatchLen, lastBatchLen, "The size of the last batch len should match for test case %d", i)
	}

}

func TestGetLast(t *testing.T) {
	store, closeF := openBadgerStore(t)
	defer closeF()

	// insert
	numElems := uint64(20)
	prefixes := [][]byte{{storage.IndexPrefix}, {storage.HistoryCachePrefix}, {storage.HyperCachePrefix}}
	for _, prefix := range prefixes {
		for i := uint64(0); i < numElems; i++ {
			key := util.Uint64AsBytes(i)
			store.Mutate([]*storage.Mutation{
				{prefix[0], key, key},
			})
		}
	}

	// get last element for history prefix
	kv, err := store.GetLast(storage.HistoryCachePrefix)
	require.NoError(t, err)
	require.Equalf(t, util.Uint64AsBytes(numElems-1), kv.Key, "The key should match the last inserted element")
	require.Equalf(t, util.Uint64AsBytes(numElems-1), kv.Value, "The value should match the last inserted element")
}

func TestBackupLoad(t *testing.T) {
	store, closeF := openBadgerStore(t)
	defer closeF()

	// insert
	numElems := uint64(20)
	prefixes := [][]byte{{storage.IndexPrefix}, {storage.HistoryCachePrefix}, {storage.HyperCachePrefix}}
	for _, prefix := range prefixes {
		for i := uint64(0); i < numElems; i++ {
			key := util.Uint64AsBytes(i)
			store.Mutate([]*storage.Mutation{
				{prefix[0], key, key},
			})
		}
	}

	version, err := store.GetLastVersion()
	require.NoError(t, err)

	backupDir := mustTempDir()
	defer os.RemoveAll(backupDir)
	backupFile, err := os.Create(filepath.Join(backupDir, "backup"))
	require.NoError(t, err)

	store.Backup(backupFile, version)

	restore, recloseF := openBadgerStore(t)
	defer recloseF()
	restore.Load(backupFile)
	reversion, err := store.GetLastVersion()

	require.NoError(t, err)
	require.Equal(t, reversion, version, "Error in restored version")
}

func BenchmarkMutate(b *testing.B) {
	store, closeF := openBadgerStore(b)
	defer closeF()
	prefix := byte(0x0)
	b.N = 100000
	b.ResetTimer()

	t := metrics.NewTimer()
	metrics.Register("mutations", t)
	go metrics.Log(metrics.DefaultRegistry, 20*time.Second, log.New(os.Stderr, "metrics: ", log.Lmicroseconds))

	for i := 0; i < b.N; i++ {
		t.Time(func() {
			store.Mutate([]*storage.Mutation{
				{prefix, rand.Bytes(128), []byte("Value")},
			})
		})
	}

}

func BenchmarkGet(b *testing.B) {
	store, closeF := openBadgerStore(b)
	defer closeF()
	prefix := byte(0x0)
	N := 10000
	b.N = N
	var key []byte

	// populate storage
	for i := 0; i < N; i++ {
		if i == 10 {
			key = rand.Bytes(128)
			store.Mutate([]*storage.Mutation{
				{prefix, key, []byte("Value")},
			})
		} else {
			store.Mutate([]*storage.Mutation{
				{prefix, rand.Bytes(128), []byte("Value")},
			})
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		store.Get(prefix, key)
	}

}

func BenchmarkGetRangeInLargeTree(b *testing.B) {
	store, closeF := openBadgerStore(b)
	defer closeF()
	prefix := byte(0x0)
	N := 1000000

	// populate storage
	for i := 0; i < N; i++ {
		store.Mutate([]*storage.Mutation{
			{prefix, []byte{byte(i)}, []byte("Value")},
		})
	}

	b.ResetTimer()

	b.Run("Small range", func(b *testing.B) {
		b.N = 10000
		for i := 0; i < b.N; i++ {
			store.GetRange(prefix, []byte{10}, []byte{10})
		}
	})

	b.Run("Large range", func(b *testing.B) {
		b.N = 10000
		for i := 0; i < b.N; i++ {
			store.GetRange(prefix, []byte{10}, []byte{35})
		}
	})

}

func openBadgerStore(t require.TestingT) (*BadgerStore, func()) {
	path := mustTempDir()

	store, err := NewBadgerStore(filepath.Join(path, "backupbadger_store_test.db"))
	if err != nil {
		t.Errorf("Error opening badger store: %v", err)
		t.FailNow()
	}
	return store, func() {
		store.Close()
		defer os.RemoveAll(path)
	}
}

func mustTempDir() string {
	var err error
	path, err := ioutil.TempDir("", "backup-test-")
	if err != nil {
		panic("failed to create temp dir")
	}
	return path
}

func deleteFile(path string) {
	err := os.RemoveAll(path)
	if err != nil {
		fmt.Printf("Unable to remove db file %s", err)
	}
}
