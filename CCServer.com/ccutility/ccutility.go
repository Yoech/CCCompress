package ccutility

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"os"

	// xxx
	_ "net/http/pprof"
	"strings"
	"time"
)

// GetAllFileByExt .
func GetAllFileByExt(pathname string, ext string, s []string) ([]string, error) {
	rd, err := ioutil.ReadDir(pathname)
	if err != nil {
		log.Printf("GetAllFileByExt.ReadDir[%v].err[%v]", pathname, err)
		return s, err
	}
	for _, fi := range rd {
		if fi.IsDir() {
			fullDir := pathname + "/" + fi.Name()
			s, err = GetAllFileByExt(fullDir, ext, s)
			if err != nil {
				log.Printf("GetAllFileByExt[%v].err[%v]", fullDir, err)
				return s, err
			}
		} else {
			if strings.HasSuffix(strings.ToLower(fi.Name()), strings.ToLower(ext)) {
				fullName := pathname + "/" + fi.Name()
				s = append(s, fullName)
			}
		}
	}
	return s, nil
}

// PathExists .
func PathExists(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err == nil {
		log.Printf("fileInfo[%v].size[%v]", path, fileInfo.Size())
		return true, nil
	}
	if !os.IsNotExist(err) {
		return true, nil
	}
	return false, err
}

// Round .
func Round(f float64) int {
	return int(math.Floor(f + 0.5))
}

// ReplaceSlash a://b/c -> a:b:c
func ReplaceSlash(uri string) (ret string) {
	ret = strings.Replace(uri, "//", "", -1)
	ret = strings.Replace(ret, "/", ":", -1)
	return ret
}

// RemoveLastSlash trim the last chr('/')
func RemoveLastSlash(uri string) (ret string) {
	ret = uri
	if ret[len(ret)-1:] == "/" {
		ret = ret[0 : len(ret)-1]
	}
	return ret
}

// WorkerTimer .
func WorkerTimer(d time.Duration, f func()) {
	go func(d time.Duration) {
		for {
			f()
			now := time.Now()
			next := now.Add(d)
			t := time.NewTimer(next.Sub(now))
			<-t.C
		}
	}(d)
}

// ReadBinary .
func ReadBinary(filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("ReadBinary[%v].Open.err[%v]", filePath, err)
	}

	defer file.Close()

	buf := bytes.Buffer{}

	_, err = io.Copy(&buf, file)
	if err != nil {
		return nil, fmt.Errorf("ReadBinary[%v].Read.err[%v]", filePath, err)
	}

	if err = file.Close(); err != nil {
		return nil, fmt.Errorf("ReadBinary[%v].Close.err[%v]", filePath, err)
	}

	return buf.Bytes(), nil
}

// WriteBinary .
func WriteBinary(filePath string, src []byte) (int64, error) {
	path := strings.Split(filePath, "/")
	dirs := strings.Replace(strings.Trim(fmt.Sprint(path[:len(path)-1]), "[]"), " ", "/", -1) + "/"
	name := path[len(path)-1]

	err := os.MkdirAll(dirs, os.ModePerm)
	if err != nil {
		log.Printf("WriteBinary.MkdirAll[%v].err[%v]", dirs, err)
		return 0, fmt.Errorf("WriteBinary.MkdirAll[%v].err[%v]", dirs, err)
	}

	fs, err := os.Create(dirs + name)
	if err != nil {
		log.Printf("WriteBinary.Create[%v].err[%v]", dirs+name, err)
		return 0, fmt.Errorf("WriteBinary.Create[%v].err[%v]", dirs+name, err)
	}

	var dlen int64
	dlen, err = io.Copy(fs, bytes.NewReader(src))
	if err != nil {
		log.Printf("WriteBinary.Copy[%v].err[%v]", dirs+name, err)
		return 0, fmt.Errorf("WriteBinary.Copy[%v].err[%v]", dirs+name, err)
	}

	err = fs.Close()
	if err != nil {
		log.Printf("WriteBinary.Close[%v].err[%v]", dirs+name, err)
		return 0, fmt.Errorf("WriteBinary.Close[%v].err[%v]", dirs+name, err)
	}

	return dlen, nil
}

// Int64ToBytes .
func Int64ToBytes(i int64) []byte {
	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(i))
	return buf
}

// BytesToInt64 .
func BytesToInt64(buf []byte) int64 {
	return int64(binary.BigEndian.Uint64(buf))
}
