package api

import "io"

// byteReader is an extremely simple io.ByteReader that
// simply reads one byte at a time. No buffering, nothing
// fancy.
type byteReader struct {
	r io.Reader
}

func newByteReader(r io.Reader) *byteReader {
	return &byteReader{r: r}
}

func (b *byteReader) ReadByte() (byte, error) {
	sb := make([]byte, 1)
	_, err := b.r.Read(sb)
	return sb[0], err
}
