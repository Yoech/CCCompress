package cccompress

import (
	"bytes"
	"github.com/dsnet/compress/bzip2"
	"io/ioutil"
)

// CCBz2 .
type CCBz2 struct {
	CompressionLevel int
}

// Compress .
func (p *CCBz2) Compress(in []byte) ([]byte, error) {
	var (
		buffer bytes.Buffer
		out    []byte
		err    error
	)
	writer, err := bzip2.NewWriter(&buffer, &bzip2.WriterConfig{
		Level: p.CompressionLevel})
	if err != nil {
		return out, err
	}
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
func (p *CCBz2) Decompress(in []byte) ([]byte, error) {
	reader, err := bzip2.NewReader(bytes.NewReader(in), nil)
	if err != nil {
		var out []byte
		return out, err
	}
	defer reader.Close()

	return ioutil.ReadAll(reader)
}

// NewBz2 .
func NewBz2() *CCBz2 {
	return &CCBz2{
		CompressionLevel: bzip2.DefaultCompression,
	}
}

// DefaultBz2 .
var DefaultBz2 = NewBz2()
