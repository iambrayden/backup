package chunker

import (
	"io"

	"github.com/zeebo/blake3"
)

const (
	MinSize = 256 << 10
	AvgSize = 1 << 20
	MaxSize = 4 << 20

	maskS = uint64(0x0000000000007FFF)
	maskL = uint64(0x000000000000FFFF)
)

var gearTable = buildGearTable()

func buildGearTable() [256]uint64 {
	var t [256]uint64
	a := uint64(0x853C49E6748FEA9B)
	b := uint64(0xDA3E39CB94B95BDB)
	c := uint64(0xAD8E17A4E4B96C7A)
	n := uint64(1)
	for i := range t {
		x := a + b + n
		n++
		a = b ^ (b >> 11)
		b = c + (c << 3)
		c = ((c << 24) | (c >> 40)) + x
		t[i] = x
	}
	return t
}

type Chunk struct {
	Data []byte
	Hash [32]byte
}

type Chunker struct {
	r    io.Reader
	buf  []byte
	pos  int
	n    int
	done bool
}

func New(r io.Reader) *Chunker {
	return &Chunker{
		r:   r,
		buf: make([]byte, MaxSize*2),
	}
}

func (c *Chunker) Next() (*Chunk, error) {
	for {
		if c.n-c.pos < MaxSize && !c.done {
			copy(c.buf, c.buf[c.pos:c.n])
			c.n -= c.pos
			c.pos = 0
			nr, err := io.ReadFull(c.r, c.buf[c.n:])
			if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
				return nil, err
			}
			c.n += nr
			if err == io.ErrUnexpectedEOF || err == io.EOF {
				c.done = true
			}
		}

		available := c.n - c.pos
		if available == 0 {
			return nil, io.EOF
		}

		boundary := c.findBoundary(c.buf[c.pos : c.pos+available])
		chunk := &Chunk{
			Data: make([]byte, boundary),
		}
		copy(chunk.Data, c.buf[c.pos:c.pos+boundary])
		chunk.Hash = blake3.Sum256(chunk.Data)
		c.pos += boundary
		return chunk, nil
	}
}

func (c *Chunker) findBoundary(data []byte) int {
	n := len(data)
	if n <= MinSize {
		return n
	}

	var h uint64
	i := MinSize
	for ; i < n; i++ {
		h = (h>>1)^(h<<63) ^ gearTable[data[i]]
		if i < AvgSize/2 {
			if (h & maskS) == 0 {
				return i + 1
			}
		} else if i < AvgSize*2 {
			if (h & maskL) == 0 {
				return i + 1
			}
		}
		if i+1 >= MaxSize {
			return i + 1
		}
	}
	return n
}

func ChunkReader(r io.Reader) ([]Chunk, error) {
	c := New(r)
	var chunks []Chunk
	for {
		ch, err := c.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, *ch)
	}
	return chunks, nil
}
