package cccompress

import (
	"bytes"
	"compress/zlib"
	"io/ioutil"
)

// CCZlib .
type CCZlib struct {
	CompressionLevel int
}

// Compress .
func (p *CCZlib) Compress(in []byte) ([]byte, error) {
	var (
		buffer bytes.Buffer
		out    []byte
		err    error
	)
	writer, err := zlib.NewWriterLevel(&buffer, p.CompressionLevel)
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
func (p *CCZlib) Decompress(in []byte) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(in))
	if err != nil {
		var out []byte
		return out, err
	}
	defer reader.Close()

	//var bb bytes.Buffer
	//io.Copy(&bb, reader)

	return ioutil.ReadAll(reader)
}

// NewZlib .
func NewZlib() *CCZlib {
	return &CCZlib{
		CompressionLevel: zlib.DefaultCompression,
	}
}

// DefaultZlib .
var DefaultZlib = NewZlib()
