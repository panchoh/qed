package rocks

import (
	"bytes"
	"time"

	"github.com/bbva/qed/storage"
	metrics "github.com/rcrowley/go-metrics"
	"github.com/tecbot/gorocksdb"
)

type Stats struct {
	mutations    metrics.Timer
	rangeQueries metrics.Timer
}

type RocksDBStore struct {
	db    *gorocksdb.DB
	stats *Stats
}

type Options struct {
	Path string
}

func NewRocksDBStore(path string) (*RocksDBStore, error) {
	return NewRocksDBStoreOpts(&Options{Path: path})
}

func NewRocksDBStoreOpts(opts *Options) (*RocksDBStore, error) {
	options := gorocksdb.NewDefaultOptions()
	options.SetCreateIfMissing(true)
	options.IncreaseParallelism(4)

	blockOpts := gorocksdb.NewDefaultBlockBasedTableOptions()
	blockOpts.SetFilterPolicy(gorocksdb.NewBloomFilter(10))
	options.SetBlockBasedTableFactory(blockOpts)
	//options.SetEnv()
	//options.SetPrefixExtractor

	db, err := gorocksdb.OpenDb(options, opts.Path)
	if err != nil {
		return nil, err
	}

	stats := &Stats{
		mutations:    metrics.NewTimer(),
		rangeQueries: metrics.NewTimer(),
	}
	metrics.Register("badger.mutate", stats.mutations)
	metrics.Register("badger.get_range", stats.rangeQueries)

	store := &RocksDBStore{
		db:    db,
		stats: stats,
	}

	return store, nil
}

func (s RocksDBStore) Mutate(mutations []*storage.Mutation) error {
	ts := time.Now()
	batch := gorocksdb.NewWriteBatch()
	defer batch.Destroy()
	for _, m := range mutations {
		key := append([]byte{m.Prefix}, m.Key...)
		switch {
		case m.Type == storage.Delete:
			batch.Delete(key)
		case m.Type == storage.Set:
			fallthrough
		default:
			batch.Put(key, m.Value)
		}
	}
	err := s.db.Write(gorocksdb.NewDefaultWriteOptions(), batch)
	s.stats.mutations.Update(time.Since(ts))
	return err
}

func (s RocksDBStore) Get(prefix byte, key []byte) (*storage.KVPair, error) {
	result := new(storage.KVPair)
	result.Key = key
	k := append([]byte{prefix}, key...)
	v, err := s.db.GetBytes(gorocksdb.NewDefaultReadOptions(), k)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, storage.ErrKeyNotFound
	}
	result.Value = v
	return result, nil
}

func (s RocksDBStore) GetRange(prefix byte, start, end []byte) (storage.KVRange, error) {

	ts := time.Now()

	result := make(storage.KVRange, 0)
	startKey := append([]byte{prefix}, start...)
	endKey := append([]byte{prefix}, end...)
	it := s.db.NewIterator(gorocksdb.NewDefaultReadOptions())
	defer it.Close()
	for it.Seek(startKey); it.Valid(); it.Next() {
		key := it.Key().Data()
		if bytes.Compare(key, endKey) > 0 {
			break
		}
		value := it.Value().Data()
		result = append(result, storage.KVPair{key[1:], value})
	}

	s.stats.rangeQueries.Update(time.Since(ts))

	return result, nil
}

func (s RocksDBStore) GetLast(prefix byte) (*storage.KVPair, error) {
	it := s.db.NewIterator(gorocksdb.NewDefaultReadOptions())
	defer it.Close()
	it.SeekForPrev([]byte{prefix, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})
	if it.ValidForPrefix([]byte{prefix}) {
		result := new(storage.KVPair)
		key := it.Key().Data()
		result.Key = key[1:]
		result.Value = it.Value().Data()
		return result, nil
	}
	return nil, storage.ErrKeyNotFound
}

type RocksDBKVPairReader struct {
	prefix byte
	it     *gorocksdb.Iterator
}

func NewRocksDBKVPairReader(prefix byte, db *gorocksdb.DB) *RocksDBKVPairReader {
	opts := gorocksdb.NewDefaultReadOptions()
	opts.SetFillCache(false)
	it := db.NewIterator(opts)
	it.Seek([]byte{prefix})
	return &RocksDBKVPairReader{prefix, it}
}

func (r *RocksDBKVPairReader) Read(buffer []*storage.KVPair) (n int, err error) {
	for n = 0; r.it.ValidForPrefix([]byte{r.prefix}) && n < len(buffer); r.it.Next() {
		key := r.it.Key().Data()
		value := r.it.Value().Data()
		buffer[n] = &storage.KVPair{key[1:], value}
		n++
	}

	// TODO should i close the iterator and transaction?
	return n, err
}

func (r *RocksDBKVPairReader) Close() {
	r.it.Close()
}

func (s RocksDBStore) GetAll(prefix byte) storage.KVPairReader {
	return NewRocksDBKVPairReader(prefix, s.db)
}

func (s RocksDBStore) Close() error {
	s.db.Close()
	return nil
}
