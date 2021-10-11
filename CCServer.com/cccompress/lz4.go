package cccompress

import (
	"bytes"
	"fmt"
	"github.com/pierrec/lz4"
	"io/ioutil"
)

// CCLz4 .
type CCLz4 struct {
	CompressionLevel int
}

// Compress .
func (p *CCLz4) Compress(in []byte) ([]byte, error) {
	var (
		buffer bytes.Buffer
		out    []byte
		err    error
	)
	writer := lz4.NewWriter(&buffer)
	if writer == nil {
		return out, fmt.Errorf("CCLz4.Compress.NewWriter.nil")
	}
	writer.Header.CompressionLevel = p.CompressionLevel
	_, err = writer.Write(in)
	if err != nil {
		return out, err
	}
	if err = writer.Close(); err != nil {
		return out, err
	}
	return buffer.Bytes(), nil
}

// Decompress .
func (p *CCLz4) Decompress(in []byte) ([]byte, error) {
	reader := lz4.NewReader(bytes.NewReader(in))
	if reader == nil {
		var out []byte
		return out, fmt.Errorf("CCLz4.Decompress.NewReader.nil")
	}

	return ioutil.ReadAll(reader)
}

// NewLz4 .
func NewLz4() *CCLz4 {
	return &CCLz4{
		CompressionLevel: 9,
	}
}

// DefaultLz4 .
var DefaultLz4 = NewLz4()
