package cccompress

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/Yoech/CCCompress/ccutility"
	"log"
	"math"
	"os"
	"strings"
	"sync"
)

var cccompressFormat = [...]byte{0x00, 0x00, 0x43, 0x43}
var cccompressVersion = []byte{'1', '0', '1', '0', '9', '0', '5'}

// compressed mode
const (
	Uncompressed = 0
	GZip         = 1
	Zlib         = 2
	Bz2          = 3
	Lzw          = 4
	Lz4          = 5
)

// TagCCHeaderInfo .
type TagCCHeaderInfo struct {
	Format        [4]byte // 0x00 0x00 0x43 0x43
	Version       [7]byte // 1050905
	CompressMode  [1]byte // 0=Uncompressed、1=GZip、2=Zlib、3=Bz2、4=Lzw、5=Lz4
	CompressedLen [8]byte // Length of compressed data.range:[0x00,0xFFFFFFFFFFFFFFFF]
	OriginLen     [8]byte // Length before data compression.range:[0x00,0xFFFFFFFFFFFFFFFF]
}

// IsValidFormat .
func IsValidFormat(s []byte) bool {
	return bytes.Equal(s, []byte{0x00, 0x00, 0x43, 0x43})
}

// IsValidCompressMode .
func IsValidCompressMode(s byte) bool {
	if s < Uncompressed || s > Lz4 {
		return false
	}
	return true
}

// IsValid .
func (p *TagCCHeaderInfo) IsValid() bool {
	if p == nil {
		return false
	}

	b := make([]byte, 4)
	copy(b, p.Format[:])
	if !IsValidFormat(b) {
		return false
	}

	if !IsValidCompressMode(p.CompressMode[0]) {
		return false
	}

	return true
}

// Compress .
func Compress(key string, src []byte, compressMode byte) (ret []byte, err error) {
	if src == nil {
		return nil, fmt.Errorf("Compress[%v].src nil", key)
	}

	a := strings.Split(key, ".")
	if len(a) < 2 {
		return nil, fmt.Errorf("Compress[%v].Split less", key)
	}
	x := len(a[0])
	y := len(a[1])
	sLen := len(src)

	if x < 0 || y < 0 {
		return nil, fmt.Errorf("Compress[%v].length less", key)
	}

	// if the header format is correct, we ignore it
	header, err := getHeader(src)
	if err == nil {
		return nil, fmt.Errorf("Compress[%v].header exists.ignore it", key)
	}

	var dst []byte
	switch compressMode {
	case GZip:
		dst, err = DefaultGzip.Compress(src)
		if err != nil {
			return nil, fmt.Errorf("Compress[%v].GZip.Compress.err[%v]", key, err)
		}
	case Zlib:
		dst, err = DefaultZlib.Compress(src)
		if err != nil {
			return nil, fmt.Errorf("Compress[%v].Zlib.Compress.err[%v]", key, err)
		}
	case Bz2:
		dst, err = DefaultBz2.Compress(src)
		if err != nil {
			return nil, fmt.Errorf("Compress[%v].Bz2.Compress.err[%v]", key, err)
		}
	case Lzw:
		dst, err = DefaultLzw.Compress(src)
		if err != nil {
			return nil, fmt.Errorf("Compress[%v].Lzw.Compress.err[%v]", key, err)
		}
	case Lz4:
		dst, err = DefaultLz4.Compress(src)
		if err != nil {
			return nil, fmt.Errorf("Compress[%v].Lz4.Compress.err[%v]", key, err)
		}
	default:
		dst = make([]byte, len(src))
		copy(dst, src)
	}

	total := len(dst)
	if total > 848 {
		total = 848
	}

	m, n := 0, 0
	for i := 0; i < total/2; i++ {
		dst[i*2] ^= a[0][m]
		dst[i*2+1] ^= a[1][n]
		if m < (x - 1) {
			m++
		} else {
			m = 0
		}
		if n < (y - 1) {
			n++
		} else {
			n = 0
		}
	}

	// make header
	header = &TagCCHeaderInfo{
		Format:       cccompressFormat,
		CompressMode: [...]byte{compressMode},
	}
	copy(header.Version[:], cccompressVersion)
	copy(header.CompressedLen[:], ccutility.Int64ToBytes(int64(len(dst))))
	copy(header.OriginLen[:], ccutility.Int64ToBytes(int64(sLen)))

	buf := new(bytes.Buffer)
	if err = binary.Write(buf, binary.LittleEndian, header); err != nil {
		return nil, fmt.Errorf("Compress[%v].binary.Write.err[%v]", key, err)
	}
	if _, err = buf.Write(dst); err != nil {
		return nil, fmt.Errorf("Compress[%v].Write.err[%v]", key, err)
	}

	return buf.Bytes(), nil
}

// Decompress .
func Decompress(key string, src []byte) (header *TagCCHeaderInfo, ret []byte, err error) {
	if src == nil {
		return nil, nil, fmt.Errorf("Decompress[%v].src.nil", key)
	}

	a := strings.Split(key, ".")
	if len(a) < 2 {
		return nil, nil, fmt.Errorf("Decompress[%v].Split less", key)
	}
	x := len(a[0])
	y := len(a[1])

	if x < 0 || y < 0 {
		return nil, nil, fmt.Errorf("Decompress[%v].length less", key)
	}

	// if the header format isn't correct, we ignore it
	header, err = getHeader(src)
	if err != nil {
		return nil, nil, err
	}

	if header == nil {
		return nil, nil, fmt.Errorf("Decompress[%v].header.nil", key)
	}

	srcBody := src[binary.Size(header):]

	total := len(srcBody)
	if total > 848 {
		total = 848
	}

	m, n := 0, 0
	for i := 0; i < total/2; i++ {
		srcBody[i*2] ^= a[0][m]
		srcBody[i*2+1] ^= a[1][n]
		if m < (x - 1) {
			m++
		} else {
			m = 0
		}
		if n < (y - 1) {
			n++
		} else {
			n = 0
		}
	}

	var dst []byte
	compressMode := header.CompressMode[:]
	switch compressMode[0] {
	case GZip:
		dst, err = DefaultGzip.Decompress(srcBody)
		if err != nil {
			return nil, nil, fmt.Errorf("Decompress[%v].GZip.Compress.err[%v]", key, err)
		}
	case Zlib:
		dst, err = DefaultZlib.Decompress(srcBody)
		if err != nil {
			return nil, nil, fmt.Errorf("Decompress[%v].Zlib.Compress.err[%v]", key, err)
		}
	case Bz2:
		dst, err = DefaultBz2.Decompress(srcBody)
		if err != nil {
			return nil, nil, fmt.Errorf("Decompress[%v].Bz2.Compress.err[%v]", key, err)
		}
	case Lzw:
		dst, err = DefaultLzw.Decompress(srcBody)
		if err != nil {
			return nil, nil, fmt.Errorf("Decompress[%v].Lzw.Compress.err[%v]", key, err)
		}
	case Lz4:
		dst, err = DefaultLz4.Decompress(srcBody)
		if err != nil {
			return nil, nil, fmt.Errorf("Decompress[%v].Lz4.Compress.err[%v]", key, err)
		}
	default:
		dst = make([]byte, len(srcBody))
		copy(dst, srcBody)
	}

	return header, dst, nil
}

// getHeader .
func getHeader(src []byte) (header *TagCCHeaderInfo, err error) {
	header = &TagCCHeaderInfo{}
	if len(src) < binary.Size(header) {
		return nil, fmt.Errorf("getHeader.src.nil")
	}

	buf := bytes.NewBuffer(src)

	if err := binary.Read(buf, binary.LittleEndian, header); err != nil {
		return nil, fmt.Errorf("getHeader.Read.err[%v]", err)
	}

	if !header.IsValid() {
		return nil, fmt.Errorf("getHeader.header.IsValid.false")
	}

	l := ccutility.BytesToInt64(header.CompressedLen[:])

	bodySize := len(src) - binary.Size(header)
	if int(l) != bodySize {
		return nil, fmt.Errorf("getHeader.size[%v/%v].no match", l, bodySize)
	}

	return header, nil
}

// CompressFile .
func CompressFile(filePath string, key string, compressMode int, bOverWrite bool) (dlen int64, err error) {
	src, err := ccutility.ReadBinary(filePath)
	if err != nil {
		return 0, fmt.Errorf("CompressFile[%v].ReadBinary.err[%v]", filePath, err)
	}
	dst, err := Compress(key, src, byte(compressMode))
	if err != nil {
		return 0, fmt.Errorf("CompressFile[%v].Compress.err[%v]", filePath, err)
	}
	if !bOverWrite {
		os.Rename(filePath, filePath+".bak")
	}
	return ccutility.WriteBinary(filePath, dst)
}

// DecompressFile .
func DecompressFile(filePath string, key string, bOverWrite bool) (dlen int64, err error) {
	src, err := ccutility.ReadBinary(filePath)
	if err != nil {
		return 0, fmt.Errorf("DecompressFile[%v].ReadBinary.err[%v]", filePath, err)
	}
	_, dst, err := Decompress(key, src)
	if err != nil {
		return 0, fmt.Errorf("DecompressFile[%v].Decompress.err[%v]", filePath, err)
	}
	if !bOverWrite {
		os.Rename(filePath, filePath+".bak")
	}
	return ccutility.WriteBinary(filePath, dst)
}

// CompressFolders .
func CompressFolders(folders string, ext string, key string, compressMode int, bOverWrite bool, iWorkerNum int) (successed int64, err error) {
	var allFile []string
	allFile, err = ccutility.GetAllFileByExt(folders, ext, allFile)
	if err != nil {
		return 0, err
	}

	successed = 0

	total := len(allFile)
	pagePerCPU := 1

	if total > iWorkerNum {
		f := math.Ceil(float64(total) / float64(iWorkerNum))
		pagePerCPU = ccutility.Round(f)
	} else {
		iWorkerNum = total
		pagePerCPU = 1
	}

	ch := make(chan int, iWorkerNum)
	var wg = &sync.WaitGroup{}
	var lock = new(sync.RWMutex)

	for i := 0; i < iWorkerNum; i++ {
		wg.Add(1)
		go func(ch <-chan int, wg *sync.WaitGroup, i int, t int, p int, f []string, k string, m int, w bool) {
			defer wg.Done()
			for idx := i * p; idx < (i+1)*p; idx++ {
				if idx >= t {
					break
				}
				if _, err = CompressFile(f[idx], k, m, w); err == nil {
					lock.Lock()
					successed++
					lock.Unlock()
				} else {
					log.Printf("f[%v].err=%v", f[idx], err)
				}
			}
		}(ch, wg, i, total, pagePerCPU, allFile, key, compressMode, bOverWrite)
	}
	wg.Wait()
	return successed, err
}

// DecompressFolders .
func DecompressFolders(folders string, ext string, key string, bOverWrite bool, iWorkerNum int) (successed int64, err error) {
	var allFile []string
	allFile, err = ccutility.GetAllFileByExt(folders, ext, allFile)
	if err != nil {
		return 0, err
	}
	successed = 0

	total := len(allFile)
	pagePerCPU := 1

	if total > iWorkerNum {
		f := math.Ceil(float64(total) / float64(iWorkerNum))
		pagePerCPU = ccutility.Round(f)
	} else {
		iWorkerNum = total
		pagePerCPU = 1
	}

	ch := make(chan int, iWorkerNum)
	var wg = &sync.WaitGroup{}
	var lock = new(sync.RWMutex)

	for i := 0; i < iWorkerNum; i++ {
		wg.Add(1)
		go func(ch <-chan int, wg *sync.WaitGroup, i int, t int, p int, f []string, k string, w bool) {
			defer wg.Done()
			for idx := i * p; idx < (i+1)*p; idx++ {
				if idx >= t {
					break
				}
				if _, err = DecompressFile(f[idx], k, w); err == nil {
					lock.Lock()
					successed++
					lock.Unlock()
				}
			}
		}(ch, wg, i, total, pagePerCPU, allFile, key, bOverWrite)
	}
	wg.Wait()
	return successed, err
}
