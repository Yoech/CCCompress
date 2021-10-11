package cccompress

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
)

// CCGzip .
type CCGzip struct {
	CompressionLevel int
}

// Compress .
func (p *CCGzip) Compress(in []byte) ([]byte, error) {
	var (
		buffer bytes.Buffer
		out    []byte
		err    error
	)
	writer, err := gzip.NewWriterLevel(&buffer, p.CompressionLevel)
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
func (p *CCGzip) Decompress(in []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(in))
	if err != nil {
		var out []byte
		return out, err
	}
	defer reader.Close()

	return ioutil.ReadAll(reader)
}

// NewGzip .
func NewGzip() *CCGzip {
	return &CCGzip{
		CompressionLevel: gzip.DefaultCompression,
	}
}

// DefaultGzip .
var DefaultGzip = NewGzip()
