package main

import (
	"CCServer.com/cccompress"
	"flag"
	"log"
	"os"
	"time"
)

var (
	bCompress   bool
	bDecompress bool
	bOverWrite  bool
	iMode       int
	iWorkerNum  int
	sTarget     string
	sExt        string
	sKey        string
)

func init() {
	flag.BoolVar(&bCompress, "c", false, "Compress")
	flag.BoolVar(&bDecompress, "d", false, "Decompress")
	flag.BoolVar(&bOverWrite, "w", false, "Overwrite origin files,otherwise rename origin files to .bak")
	flag.IntVar(&iMode, "m", cccompress.Uncompressed, "Compress/Decompress mode")
	flag.IntVar(&iWorkerNum, "n", 10, "Number of workers when compress/decompress folders")
	flag.StringVar(&sTarget, "t", "", "Target path")
	flag.StringVar(&sExt, "e", "", "Ext")
	flag.StringVar(&sKey, "k", "", "Obfuscation key")

	flag.Usage = useAge
}

// useAge .
func useAge() {
	cmdStr := "\n*****************************************\n"
	cmdStr += "Usage:\n"
	cmdStr += "*****************************************\n"
	log.Printf(cmdStr)

	flag.PrintDefaults()
}

func main() {
	flag.Parse()

	// check test mode
	params := os.Args
	if len(params) == 1 {
		useAge()
		return
	}

	s := time.Now()
	var total int64
	var fi os.FileInfo
	var err error

	fi, err = os.Stat(sTarget)
	if err != nil {
		useAge()
		return
	}

	if fi.IsDir() {
		if bCompress {
			total, err = cccompress.CompressFolders(sTarget, sExt, sKey, iMode, bOverWrite, iWorkerNum)
		} else {
			total, err = cccompress.DecompressFolders(sTarget, sExt, sKey, iMode, bOverWrite, iWorkerNum)
		}
	} else {
		if bCompress {
			total, err = cccompress.CompressFile(sTarget, sKey, iMode, bOverWrite)
		} else {
			total, err = cccompress.DecompressFile(sTarget, sKey, iMode, bOverWrite)
		}
	}

	cost := time.Now().Unix() - s.Unix()
	log.Printf("Total[%v].finished!...cost[%v s].err[%v]", total, cost, err)
	return
}
