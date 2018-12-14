package hyper2

import (
	"expvar"
	"os"
	"testing"
	"time"

	"github.com/bbva/qed/hashing"
	"github.com/bbva/qed/log"
	"github.com/bbva/qed/testutils/rand"
	storage_utils "github.com/bbva/qed/testutils/storage"
	metrics "github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdd(t *testing.T) {

	log.SetLogger("TestAdd", log.SILENT)

	testCases := []struct {
		eventDigest      hashing.Digest
		expectedRootHash hashing.Digest
	}{
		{hashing.Digest{0x0}, hashing.Digest{0x0}},
		{hashing.Digest{0x1}, hashing.Digest{0x1}},
		{hashing.Digest{0x2}, hashing.Digest{0x3}},
		{hashing.Digest{0x3}, hashing.Digest{0x0}},
		{hashing.Digest{0x4}, hashing.Digest{0x4}},
		{hashing.Digest{0x5}, hashing.Digest{0x1}},
		{hashing.Digest{0x6}, hashing.Digest{0x7}},
		{hashing.Digest{0x7}, hashing.Digest{0x0}},
		{hashing.Digest{0x8}, hashing.Digest{0x8}},
		{hashing.Digest{0x9}, hashing.Digest{0x1}},
	}

	store, closeF := storage_utils.OpenBPlusTreeStore()
	defer closeF()
	simpleCache := NewFastCache(1000 * 68)
	tree := NewHyperTree(hashing.NewFakeXorHasher, store, simpleCache)

	for i, c := range testCases {
		index := uint64(i)
		snapshot, mutations, err := tree.Add(c.eventDigest, index)
		tree.store.Mutate(mutations)
		require.NoErrorf(t, err, "This should not fail for index %d", i)
		assert.Equalf(t, c.expectedRootHash, snapshot, "Incorrect root hash for index %d", i)

	}
}

func BenchmarkAdd(b *testing.B) {

	log.SetLogger("BenchmarkAdd", log.SILENT)

	store, closeF := storage_utils.OpenBadgerStore(b, "/var/tmp/hyper_tree_test.db")
	defer closeF()

	hasher := hashing.NewSha256Hasher()
	fastCache := NewFastCache(CacheSize)
	//freeCache := cache.NewFreeCache((1 << 26) * 100)
	tree := NewHyperTree(hashing.NewSha256Hasher, store, fastCache)

	t := metrics.NewTimer()
	metrics.Register("hyper.test_add", t)

	reg := metrics.NewPrefixedChildRegistry(metrics.DefaultRegistry, "store.")
	storeNumReads := metrics.NewGauge()
	storeNumWrites := metrics.NewGauge()
	storeBytesRead := metrics.NewGauge()
	storeBytesWritten := metrics.NewGauge()
	storeGets := metrics.NewGauge()
	storePuts := metrics.NewGauge()
	storeBlockedPuts := metrics.NewGauge()
	storeNumMemtableGets := metrics.NewGauge()
	cacheSize := metrics.NewGauge()
	reg.Register("disk_reads_total", storeNumReads)
	reg.Register("disk_writes_total", storeNumWrites)
	reg.Register("read_bytes", storeBytesRead)
	reg.Register("written_bytes", storeBytesWritten)
	reg.Register("gets_total", storeGets)
	reg.Register("puts_total", storePuts)
	reg.Register("blocked_puts_total", storeBlockedPuts)
	reg.Register("memtable_gets_total", storeNumMemtableGets)
	metrics.Register("cache.size", cacheSize)

	metrics.RegisterDebugGCStats(metrics.DefaultRegistry)
	metrics.RegisterRuntimeMemStats(metrics.DefaultRegistry)

	f, _ := os.Create("/var/tmp/stats3")
	defer f.Close()

	go func() {
		for _ = range time.Tick(1 * time.Minute) {

			storeNumReads.Update(expvar.Get("badger_disk_reads_total").(*expvar.Int).Value())
			storeNumWrites.Update(expvar.Get("badger_disk_writes_total").(*expvar.Int).Value())
			storeBytesRead.Update(expvar.Get("badger_read_bytes").(*expvar.Int).Value())
			storeBytesWritten.Update(expvar.Get("badger_written_bytes").(*expvar.Int).Value())
			storeGets.Update(expvar.Get("badger_gets_total").(*expvar.Int).Value())
			storePuts.Update(expvar.Get("badger_puts_total").(*expvar.Int).Value())
			storeBlockedPuts.Update(expvar.Get("badger_blocked_puts_total").(*expvar.Int).Value())
			storeNumMemtableGets.Update(expvar.Get("badger_memtable_gets_total").(*expvar.Int).Value())
			expvar.Get("badger_lsm_level_gets_total").(*expvar.Map).Do(func(kv expvar.KeyValue) {
				m := reg.GetOrRegister("lsm_level_gets_total."+kv.Key, metrics.NewGauge())
				m.(metrics.Gauge).Update(kv.Value.(*expvar.Int).Value())
			})
			expvar.Get("badger_lsm_bloom_hits_total").(*expvar.Map).Do(func(kv expvar.KeyValue) {
				m := reg.GetOrRegister("lsm_bloom_hits_total."+kv.Key, metrics.NewGauge())
				m.(metrics.Gauge).Update(kv.Value.(*expvar.Int).Value())
			})
			expvar.Get("badger_pending_writes_total").(*expvar.Map).Do(func(kv expvar.KeyValue) {
				m := reg.GetOrRegister("pending_writes_total", metrics.NewGauge())
				m.(metrics.Gauge).Update(kv.Value.(*expvar.Int).Value())
			})
			expvar.Get("badger_lsm_size_bytes").(*expvar.Map).Do(func(kv expvar.KeyValue) {
				m := reg.GetOrRegister("lsm_size_bytes", metrics.NewGauge())
				m.(metrics.Gauge).Update(kv.Value.(*expvar.Int).Value())
			})
			cacheSize.Update(int64(fastCache.Size()))
			metrics.CaptureDebugGCStatsOnce(metrics.DefaultRegistry)
			metrics.CaptureRuntimeMemStatsOnce(metrics.DefaultRegistry)

			metrics.WriteJSONOnce(metrics.DefaultRegistry, f)
		}
	}()

	//go metrics.WriteJSON(metrics.DefaultRegistry, 1*time.Minute, f)

	b.ResetTimer()
	b.N = 20000000
	for i := 0; i < b.N; i++ {
		t.Time(func() {
			key := hasher.Do(rand.Bytes(32))
			_, mutations, _ := tree.Add(key, uint64(i))
			store.Mutate(mutations)
		})
	}

}
