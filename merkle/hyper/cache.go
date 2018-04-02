package hyper

// SimpleCache is a cache that contains the hashes of the pre-computed nodes
type SimpleCache struct {
	nodes map[[32]byte][]byte // node map containing the cached hashes
}

func (c *SimpleCache) Put(key []byte, value []byte) error {
	var aux [32]byte
	copy(aux[:], key[:32])
	c.nodes[aux] = value
	return nil
}

func (c *SimpleCache) Get(key []byte) ([]byte, bool) {
	var aux [32]byte
	copy(aux[:], key[:32])
	result, ok := c.nodes[aux]
	return result, ok
}

func (c *SimpleCache) Exists(key []byte) bool {
	var aux [32]byte
	copy(aux[:], key[:32])
	_, ok := c.nodes[aux]
	return ok
}

// NewSimpleCache creates a new cache structure, already initialized
// with a specified size
func NewSimpleCache(size uint64) *SimpleCache {
	return &SimpleCache{make(map[[32]byte][]byte, size)}
}