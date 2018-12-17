package hyper2

import (
	"fmt"

	"github.com/bbva/qed/util"
)

type Position struct {
	Index  []byte
	Height uint16

	bytes []byte
}

func NewPosition(index []byte, height uint16) *Position {
	b := make([]byte, 34) // Size of the index plus 2 bytes for the height
	copy(b, index)
	copy(b[len(index):], util.Uint16AsBytes(height))
	return &Position{
		Index:  index,
		Height: height,
		bytes:  b, // memoized id
	}
}

func (p Position) Bytes() []byte {
	return p.bytes
}

func (p Position) String() string {
	return fmt.Sprintf("Pos(%x, %d)", p.Index, p.Height)
}

func (p Position) StringId() string {
	return fmt.Sprintf("%x|%d", p.Index, p.Height)
}
