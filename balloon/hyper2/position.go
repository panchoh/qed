package hyper2

import (
	"fmt"

	"github.com/bbva/qed/util"
)

type Position struct {
	index  []byte
	height uint16
}

func NewPosition(index []byte, height uint16) *Position {
	return &Position{
		index:  index,
		height: height,
	}
}

func (p Position) Index() []byte {
	return p.index
}

func (p Position) Height() uint16 {
	return p.height
}

func (p Position) Bytes() []byte {
	b := make([]byte, 34) // Size of the index plus 2 bytes for the height
	copy(b, p.index)
	copy(b[len(p.index):], util.Uint16AsBytes(p.height))
	return b
}

func (p Position) String() string {
	return fmt.Sprintf("Pos(%x, %d)", p.index, p.height)
}

func (p Position) StringId() string {
	return fmt.Sprintf("%x|%d", p.index, p.height)
}
