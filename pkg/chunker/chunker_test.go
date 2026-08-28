package chunker_test

import (
	"bytes"
	"crypto/rand"
	"io"
	"testing"

	"github.com/waylontech/backup/pkg/chunker"
)

func TestChunkerCoverage(t *testing.T) {
	data := make([]byte, 10<<20)
	if _, err := rand.Read(data); err != nil {
		t.Fatal(err)
	}

	chunks, err := chunker.ChunkReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("ChunkReader: %v", err)
	}
	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}

	var reassembled []byte
	for _, ch := range chunks {
		reassembled = append(reassembled, ch.Data...)
	}
	if !bytes.Equal(data, reassembled) {
		t.Fatalf("reassembled data (%d bytes) != original (%d bytes)", len(reassembled), len(data))
	}
}

func TestChunkerDeterministic(t *testing.T) {
	data := make([]byte, 5<<20)
	if _, err := rand.Read(data); err != nil {
		t.Fatal(err)
	}

	chunks1, err := chunker.ChunkReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	chunks2, err := chunker.ChunkReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks1) != len(chunks2) {
		t.Fatalf("chunk count differs: %d vs %d", len(chunks1), len(chunks2))
	}
	for i := range chunks1 {
		if chunks1[i].Hash != chunks2[i].Hash {
			t.Fatalf("chunk %d hash differs", i)
		}
	}
}

func TestChunkerChunkSizes(t *testing.T) {
	data := make([]byte, 20<<20)
	if _, err := rand.Read(data); err != nil {
		t.Fatal(err)
	}

	chunks, err := chunker.ChunkReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	for i, ch := range chunks {
		if i < len(chunks)-1 {
			if len(ch.Data) < chunker.MinSize {
				t.Errorf("chunk %d too small: %d < %d", i, len(ch.Data), chunker.MinSize)
			}
		}
		if len(ch.Data) > chunker.MaxSize {
			t.Errorf("chunk %d too large: %d > %d", i, len(ch.Data), chunker.MaxSize)
		}
	}
}

func TestChunkerSmallInput(t *testing.T) {
	for _, size := range []int{0, 1, 100, 1024, chunker.MinSize - 1} {
		data := make([]byte, size)
		rand.Read(data)

		chunks, err := chunker.ChunkReader(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("size %d: %v", size, err)
		}

		var got []byte
		for _, ch := range chunks {
			got = append(got, ch.Data...)
		}
		if !bytes.Equal(data, got) {
			t.Fatalf("size %d: coverage failed", size)
		}
	}
}

func TestChunkerContentDefined(t *testing.T) {
	base := make([]byte, 5<<20)
	rand.Read(base)

	chunksA, _ := chunker.ChunkReader(bytes.NewReader(base))

	modified := make([]byte, len(base))
	copy(modified, base)
	modified[1000] ^= 0xFF

	chunksB, _ := chunker.ChunkReader(bytes.NewReader(modified))

	var matchCount int
	hashSetA := map[[32]byte]bool{}
	for _, ch := range chunksA {
		hashSetA[ch.Hash] = true
	}
	for _, ch := range chunksB {
		if hashSetA[ch.Hash] {
			matchCount++
		}
	}

	if matchCount == 0 {
		t.Error("expected some chunks to match after small modification")
	}
	minExpected := len(chunksA) - 3
	if matchCount < minExpected {
		t.Errorf("expected at least %d matching chunks, got %d", minExpected, matchCount)
	}
}

func BenchmarkChunker(b *testing.B) {
	data := make([]byte, 100<<20)
	rand.Read(data)
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chunker.ChunkReader(io.NopCloser(bytes.NewReader(data)))
	}
}
