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

package history

import (
	lg "log"
	"os"
	"testing"
	"time"

	"github.com/bbva/qed/balloon/visitor"
	"github.com/bbva/qed/hashing"
	"github.com/bbva/qed/log"
	"github.com/bbva/qed/storage/bplus"
	"github.com/bbva/qed/testutils/rand"
	storage_utils "github.com/bbva/qed/testutils/storage"
	"github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdd(t *testing.T) {

	log.SetLogger("TestAdd", log.INFO)

	testCases := []struct {
		eventDigest          hashing.Digest
		expectedRootHash     hashing.Digest
		expectedMutationsLen int
	}{
		{
			eventDigest:          hashing.Digest{0x0},
			expectedRootHash:     hashing.Digest{0x0},
			expectedMutationsLen: 1,
		},
		{
			eventDigest:          hashing.Digest{0x1},
			expectedRootHash:     hashing.Digest{0x1},
			expectedMutationsLen: 2,
		},
		{
			eventDigest:          hashing.Digest{0x2},
			expectedRootHash:     hashing.Digest{0x3},
			expectedMutationsLen: 1,
		},
		{
			eventDigest:          hashing.Digest{0x3},
			expectedRootHash:     hashing.Digest{0x0},
			expectedMutationsLen: 3,
		},
		{
			eventDigest:          hashing.Digest{0x4},
			expectedRootHash:     hashing.Digest{0x4},
			expectedMutationsLen: 1,
		},
		{
			eventDigest:          hashing.Digest{0x5},
			expectedRootHash:     hashing.Digest{0x1},
			expectedMutationsLen: 2,
		},
		{
			eventDigest:          hashing.Digest{0x6},
			expectedRootHash:     hashing.Digest{0x7},
			expectedMutationsLen: 1,
		},
		{
			eventDigest:          hashing.Digest{0x7},
			expectedRootHash:     hashing.Digest{0x0},
			expectedMutationsLen: 4,
		},
		{
			eventDigest:          hashing.Digest{0x8},
			expectedRootHash:     hashing.Digest{0x8},
			expectedMutationsLen: 1,
		},
		{
			eventDigest:          hashing.Digest{0x9},
			expectedRootHash:     hashing.Digest{0x1},
			expectedMutationsLen: 2,
		},
	}

	store := bplus.NewBPlusTreeStore()
	tree := NewHistoryTree(hashing.NewFakeXorHasher, store, 30)

	for i, c := range testCases {
		index := uint64(i)
		rootHash, mutations, err := tree.Add(c.eventDigest, index)

		require.NoError(t, err)
		assert.Equalf(t, c.expectedRootHash, rootHash, "Incorrect root hash for test case %d", i)
		assert.Equalf(t, c.expectedMutationsLen, len(mutations), "The mutations should match for test case %d", i)

		store.Mutate(mutations)
	}

}

func TestProveMembership(t *testing.T) {

	log.SetLogger("TestProveMembership", log.INFO)

	testCases := []struct {
		index, version    uint64
		eventDigest       hashing.Digest
		expectedAuditPath visitor.AuditPath
	}{
		{
			index:             0,
			version:           0,
			eventDigest:       hashing.Digest{0x0},
			expectedAuditPath: visitor.AuditPath{},
		},
		{
			index:             1,
			version:           1,
			eventDigest:       hashing.Digest{0x1},
			expectedAuditPath: visitor.AuditPath{"0|0": hashing.Digest{0x0}},
		},
		{
			index:             2,
			version:           2,
			eventDigest:       hashing.Digest{0x2},
			expectedAuditPath: visitor.AuditPath{"0|1": hashing.Digest{0x1}},
		},
		{
			index:             3,
			version:           3,
			eventDigest:       hashing.Digest{0x3},
			expectedAuditPath: visitor.AuditPath{"0|1": hashing.Digest{0x1}, "2|0": hashing.Digest{0x2}},
		},
		{
			index:             4,
			version:           4,
			eventDigest:       hashing.Digest{0x4},
			expectedAuditPath: visitor.AuditPath{"0|2": hashing.Digest{0x0}},
		},
		{
			index:             5,
			version:           5,
			eventDigest:       hashing.Digest{0x5},
			expectedAuditPath: visitor.AuditPath{"0|2": hashing.Digest{0x0}, "4|0": hashing.Digest{0x4}},
		},
		{
			index:             6,
			version:           6,
			eventDigest:       hashing.Digest{0x6},
			expectedAuditPath: visitor.AuditPath{"0|2": hashing.Digest{0x0}, "4|1": hashing.Digest{0x1}},
		},
		{
			index:             7,
			version:           7,
			eventDigest:       hashing.Digest{0x7},
			expectedAuditPath: visitor.AuditPath{"0|2": hashing.Digest{0x0}, "4|1": hashing.Digest{0x1}, "6|0": hashing.Digest{0x6}},
		},
		{
			index:             8,
			version:           8,
			eventDigest:       hashing.Digest{0x8},
			expectedAuditPath: visitor.AuditPath{"0|3": hashing.Digest{0x0}},
		},
		{
			index:             9,
			version:           9,
			eventDigest:       hashing.Digest{0x9},
			expectedAuditPath: visitor.AuditPath{"0|3": hashing.Digest{0x0}, "8|0": hashing.Digest{0x8}},
		},
		{
			index:             0,
			version:           1,
			eventDigest:       hashing.Digest{0x0},
			expectedAuditPath: visitor.AuditPath{"1|0": hashing.Digest{0x1}},
		},
		{
			index:             0,
			version:           1,
			eventDigest:       hashing.Digest{0x0},
			expectedAuditPath: visitor.AuditPath{"1|0": hashing.Digest{0x1}},
		},
		{
			index:             0,
			version:           2,
			eventDigest:       hashing.Digest{0x0},
			expectedAuditPath: visitor.AuditPath{"1|0": hashing.Digest{0x1}, "2|0": hashing.Digest{0x2}},
		},
		{
			index:             0,
			version:           3,
			eventDigest:       hashing.Digest{0x0},
			expectedAuditPath: visitor.AuditPath{"1|0": hashing.Digest{0x1}, "2|1": hashing.Digest{0x1}},
		},
		{
			index:             0,
			version:           4,
			eventDigest:       hashing.Digest{0x0},
			expectedAuditPath: visitor.AuditPath{"1|0": hashing.Digest{0x1}, "2|1": hashing.Digest{0x1}, "4|0": hashing.Digest{0x4}},
		},
		{
			index:             0,
			version:           5,
			eventDigest:       hashing.Digest{0x0},
			expectedAuditPath: visitor.AuditPath{"1|0": hashing.Digest{0x1}, "2|1": hashing.Digest{0x1}, "4|1": hashing.Digest{0x1}},
		},
		{
			index:             0,
			version:           6,
			eventDigest:       hashing.Digest{0x0},
			expectedAuditPath: visitor.AuditPath{"1|0": hashing.Digest{0x1}, "2|1": hashing.Digest{0x1}, "4|1": hashing.Digest{0x1}, "6|0": hashing.Digest{0x6}},
		},
		{
			index:             0,
			version:           7,
			eventDigest:       hashing.Digest{0x0},
			expectedAuditPath: visitor.AuditPath{"1|0": hashing.Digest{0x1}, "2|1": hashing.Digest{0x1}, "4|2": hashing.Digest{0x0}},
		},
	}

	store := bplus.NewBPlusTreeStore()
	tree := NewHistoryTree(hashing.NewFakeXorHasher, store, 30)

	for i, c := range testCases {
		_, mutations, _ := tree.Add(c.eventDigest, c.index)
		store.Mutate(mutations)

		mp, err := tree.ProveMembership(c.index, c.version)
		require.NoError(t, err)
		assert.Equalf(t, c.expectedAuditPath, mp.AuditPath(), "Incorrect audit path for index %d", i)
		assert.Equal(t, c.index, mp.Index, "The index should math")
		assert.Equal(t, c.version, mp.Version, "The version should match")
	}

}

func TestProveConsistency(t *testing.T) {

	log.SetLogger("TestProveConsistency", log.INFO)

	testCases := []struct {
		eventDigest       hashing.Digest
		expectedAuditPath visitor.AuditPath
	}{
		{
			eventDigest:       hashing.Digest{0x0},
			expectedAuditPath: visitor.AuditPath{"0|0": hashing.Digest{0x0}},
		},
		{
			eventDigest:       hashing.Digest{0x1},
			expectedAuditPath: visitor.AuditPath{"0|0": hashing.Digest{0x0}, "1|0": hashing.Digest{0x1}},
		},
		{
			eventDigest:       hashing.Digest{0x2},
			expectedAuditPath: visitor.AuditPath{"0|0": hashing.Digest{0x0}, "1|0": hashing.Digest{0x1}, "2|0": hashing.Digest{0x2}},
		},
		{
			eventDigest:       hashing.Digest{0x3},
			expectedAuditPath: visitor.AuditPath{"0|1": hashing.Digest{0x1}, "2|0": hashing.Digest{0x2}, "3|0": hashing.Digest{0x3}},
		},
		{
			eventDigest:       hashing.Digest{0x4},
			expectedAuditPath: visitor.AuditPath{"0|1": hashing.Digest{0x1}, "2|0": hashing.Digest{0x2}, "3|0": hashing.Digest{0x3}, "4|0": hashing.Digest{0x4}},
		},
		{
			eventDigest:       hashing.Digest{0x5},
			expectedAuditPath: visitor.AuditPath{"0|2": hashing.Digest{0x0}, "4|0": hashing.Digest{0x4}, "5|0": hashing.Digest{0x5}},
		},
		{
			eventDigest:       hashing.Digest{0x6},
			expectedAuditPath: visitor.AuditPath{"0|2": hashing.Digest{0x0}, "4|0": hashing.Digest{0x4}, "5|0": hashing.Digest{0x5}, "6|0": hashing.Digest{0x6}},
		},
		{
			eventDigest:       hashing.Digest{0x7},
			expectedAuditPath: visitor.AuditPath{"0|2": hashing.Digest{0x0}, "4|1": hashing.Digest{0x1}, "6|0": hashing.Digest{0x6}, "7|0": hashing.Digest{0x7}},
		},
		{
			eventDigest:       hashing.Digest{0x8},
			expectedAuditPath: visitor.AuditPath{"0|2": hashing.Digest{0x0}, "4|1": hashing.Digest{0x1}, "6|0": hashing.Digest{0x6}, "7|0": hashing.Digest{0x7}, "8|0": hashing.Digest{0x8}},
		},
		{
			eventDigest:       hashing.Digest{0x9},
			expectedAuditPath: visitor.AuditPath{"0|3": hashing.Digest{0x0}, "8|0": hashing.Digest{0x8}, "9|0": hashing.Digest{0x9}},
		},
	}

	store := bplus.NewBPlusTreeStore()
	tree := NewHistoryTree(hashing.NewFakeXorHasher, store, 30)

	for i, c := range testCases {
		index := uint64(i)
		_, mutations, err := tree.Add(c.eventDigest, index)
		require.NoError(t, err)
		store.Mutate(mutations)

		start := uint64(max(0, i-1))
		end := index
		proof, err := tree.ProveConsistency(start, end)
		require.NoError(t, err)
		assert.Equalf(t, start, proof.StartVersion, "The start version should match for test case %d", i)
		assert.Equalf(t, end, proof.EndVersion, "The start version should match for test case %d", i)
		assert.Equal(t, c.expectedAuditPath, proof.AuditPath, "Invalid audit path in test case: %d", i)
	}

}

func TestProveConsistencySameVersions(t *testing.T) {

	log.SetLogger("TestProveConsistencySameVersions", log.INFO)

	testCases := []struct {
		index             uint64
		eventDigest       hashing.Digest
		expectedAuditPath visitor.AuditPath
	}{
		{
			index:             0,
			eventDigest:       hashing.Digest{0x0},
			expectedAuditPath: visitor.AuditPath{"0|0": hashing.Digest{0x0}},
		},
		{
			index:             1,
			eventDigest:       hashing.Digest{0x1},
			expectedAuditPath: visitor.AuditPath{"0|0": hashing.Digest{0x0}, "1|0": hashing.Digest{0x1}},
		},
		{
			index:             2,
			eventDigest:       hashing.Digest{0x2},
			expectedAuditPath: visitor.AuditPath{"0|1": hashing.Digest{0x1}, "2|0": hashing.Digest{0x2}},
		},
		{
			index:             3,
			eventDigest:       hashing.Digest{0x3},
			expectedAuditPath: visitor.AuditPath{"0|1": hashing.Digest{0x1}, "2|0": hashing.Digest{0x2}, "3|0": hashing.Digest{0x3}},
		},
		{
			index:             4,
			eventDigest:       hashing.Digest{0x4},
			expectedAuditPath: visitor.AuditPath{"0|2": hashing.Digest{0x0}, "4|0": hashing.Digest{0x4}},
		},
	}

	store := bplus.NewBPlusTreeStore()
	tree := NewHistoryTree(hashing.NewFakeXorHasher, store, 30)

	for i, c := range testCases {
		_, mutations, err := tree.Add(c.eventDigest, c.index)
		require.NoError(t, err)
		store.Mutate(mutations)

		proof, err := tree.ProveConsistency(c.index, c.index)
		require.NoError(t, err)
		assert.Equalf(t, c.index, proof.StartVersion, "The start version should match for test case %d", i)
		assert.Equalf(t, c.index, proof.EndVersion, "The start version should match for test case %d", i)
		assert.Equal(t, c.expectedAuditPath, proof.AuditPath, "Invalid audit path in test case: %d", i)
	}
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func BenchmarkAdd(b *testing.B) {

	log.SetLogger("BenchmarkAdd", log.SILENT)

	store, closeF := storage_utils.OpenBadgerStore(b, "/var/tmp/history_tree_test.db")
	defer closeF()

	tree := NewHistoryTree(hashing.NewSha256Hasher, store, 300)

	tm := metrics.NewTimer()
	metrics.Register("history_add", tm)
	go metrics.Log(metrics.DefaultRegistry, 15*time.Second, lg.New(os.Stderr, "metrics: ", lg.Lmicroseconds))

	b.N = 10000000
	b.ResetTimer()
	for i := uint64(0); i < uint64(b.N); i++ {
		tm.Time(func() {
			key := rand.Bytes(64)
			_, mutations, _ := tree.Add(key, i)
			store.Mutate(mutations)
		})
	}
}
