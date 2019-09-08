package main

import (
	"flag"
	"github.com/Yoech/CCCompress/cccompress"
	"log"
	"os"
	"time"
)

var bCompress bool
var bDecompress bool
var bOverWrite bool
var iMode int
var sTarget string
var sKey string

func init() {
	flag.BoolVar(&bCompress, "c", false, "Compress")
	flag.BoolVar(&bDecompress, "d", false, "Decompress")
	flag.BoolVar(&bOverWrite, "w", false, "Overwrite origin files,otherwise rename origin files to .bak")
	flag.IntVar(&iMode, "m", cccompress.Uncompressed, "Compress/Decompress mode")
	flag.StringVar(&sTarget, "t", "", "Target path")
	flag.StringVar(&sKey, "k", "", "Obfuscation key")

	flag.Usage = useAge
}

// useAge .
func useAge() {
	cmdStr := "\n*****************************************\n"
	cmdStr += "Useage:\n"
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
			total, err = cccompress.CompressFolders(sTarget, "", sKey, iMode, bOverWrite)
		} else {
			total, err = cccompress.DecompressFolders(sTarget, "", sKey, bOverWrite)
		}
	} else {
		if bCompress {
			total, err = cccompress.CompressFile(sTarget, sKey, iMode, bOverWrite)
		} else {
			total, err = cccompress.DecompressFile(sTarget, sKey, bOverWrite)
		}
	}

	cost := time.Now().Unix() - s.Unix()
	log.Printf("Total[%v].finished!...cost[%v s].err[%v]", total, cost, err)
	return
}
