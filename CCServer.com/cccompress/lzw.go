package cccompress

import (
	"bytes"
	"compress/lzw"
	"fmt"
	"io/ioutil"
)

// CCLzw .
type CCLzw struct {
	Order     lzw.Order
	ListWidth int
}

// Compress .
func (p *CCLzw) Compress(in []byte) ([]byte, error) {
	var (
		buffer bytes.Buffer
		out    []byte
		err    error
	)
	writer := lzw.NewWriter(&buffer, p.Order, p.ListWidth)
	_, err = writer.Write(in)
	if err != nil {
		return out, fmt.Errorf("CCLzw.Compress.NewWriter.nil")
	}
	if err = writer.Close(); err != nil {
		return out, err
	}
	return buffer.Bytes(), nil
}

// Decompress .
func (p *CCLzw) Decompress(in []byte) ([]byte, error) {
	reader := lzw.NewReader(bytes.NewReader(in), p.Order, p.ListWidth)
	if reader == nil {
		var out []byte
		return out, fmt.Errorf("CCLzw.Decompress.NewReader.nil")
	}
	defer reader.Close()

	return ioutil.ReadAll(reader)
}

// NewLzw .
func NewLzw() *CCLzw {
	return &CCLzw{
		Order:     lzw.LSB,
		ListWidth: 8,
	}
}

// DefaultLzw .
var DefaultLzw = NewLzw()
