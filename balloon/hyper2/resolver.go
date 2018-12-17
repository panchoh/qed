package hyper2

type CacheResolver interface {
	ShouldBeInCache(pos *Position) bool
	ShouldCache(pos *Position) bool
	ShouldCollect(pos *Position) bool
	IsOnPath(pos *Position) bool
}

type SingleTargetedCacheResolver struct {
	numBits    uint16
	cacheLevel uint16
	targetKey  []byte
}

func NewSingleTargetedCacheResolver(numBits, cacheLevel uint16, targetKey []byte) *SingleTargetedCacheResolver {
	return &SingleTargetedCacheResolver{numBits, cacheLevel, targetKey}
}

func (r SingleTargetedCacheResolver) ShouldBeInCache(pos *Position) bool {
	return pos.Height >= r.cacheLevel && !r.IsOnPath(pos)
}

func (r SingleTargetedCacheResolver) ShouldCache(pos *Position) bool {
	return pos.Height == r.cacheLevel
}

func (r SingleTargetedCacheResolver) ShouldCollect(pos *Position) bool {
	return pos.Height == r.cacheLevel
}

/*
	This method does not reach leafs. Goes from root (bit := 0) to height=1 (bit := numbits - 1)
*/
func (r SingleTargetedCacheResolver) IsOnPath(pos *Position) bool {
	bit := r.numBits - pos.Height - 1
	return bitIsSet(r.targetKey, bit) == bitIsSet(pos.Index, bit)
}

/*
	Is bit in position 'i' set to 1?
	i   :	 2				3
	bits: [00101011]	[00101011]
	mask: [00100000]	[00010000]
			 true			false
*/
func bitIsSet(bits []byte, i uint16) bool {
	return bits[i/8]&(1<<uint(7-i%8)) != 0
}
