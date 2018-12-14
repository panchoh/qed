package hyper2

type HyperTreeNavigator struct {
	numBits uint16
}

func NewHyperTreeNavigator(numBits uint16) *HyperTreeNavigator {
	return &HyperTreeNavigator{numBits}
}

func (n HyperTreeNavigator) Root() *Position {
	index := make([]byte, n.numBits/8)
	return NewPosition(index, n.numBits)
}

func (n HyperTreeNavigator) IsLeaf(pos *Position) bool {
	return pos.Height() == 0
}

func (n HyperTreeNavigator) IsRoot(pos *Position) bool {
	return pos.Height() == n.numBits
}

func (n HyperTreeNavigator) GoToLeft(pos *Position) *Position {
	if pos.Height() == 0 {
		return nil
	}
	return NewPosition(pos.Index(), pos.Height()-1)
}

func (n HyperTreeNavigator) GoToRight(pos *Position) *Position {
	if pos.Height() == 0 {
		return nil
	}
	return NewPosition(n.splitBase(pos), pos.Height()-1)
}

func (n HyperTreeNavigator) DescendToFirst(pos *Position) *Position {
	return NewPosition(pos.Index(), 0)
}

func (n HyperTreeNavigator) DescendToLast(pos *Position) *Position {
	layer := n.numBits - pos.Height()
	base := make([]byte, n.numBits/8)
	copy(base, pos.Index())
	for bit := layer; bit < n.numBits; bit++ {
		bitSet(base, bit)
	}
	return NewPosition(base, 0)
}

func (n HyperTreeNavigator) splitBase(pos *Position) []byte {
	splitBit := n.numBits - pos.Height()
	split := make([]byte, n.numBits/8)
	copy(split, pos.Index())
	if splitBit < n.numBits {
		bitSet(split, splitBit)
	}
	return split
}

func bitSet(bits []byte, i uint16) {
	bits[i/8] |= 1 << uint(7-i%8)
}
