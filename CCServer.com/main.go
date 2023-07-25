package main

import (
	"CCServer.com/cccompress"
	"CCServer.com/ccconvert"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"time"
)

var (
	iConvert int
	iQuality int
	sSrc     string
	sDst     string

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
	flag.IntVar(&iConvert, "convert", ccconvert.Png2Jpg, "Convert [1,2](png2jpg,jpg2jpg) to jpg default:1")
	flag.IntVar(&iQuality, "q", 80, "Convert image with given quality [1,100] default:80")
	flag.StringVar(&sSrc, "src", "", "source images path")
	flag.StringVar(&sDst, "dst", "", "dest images path")

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

	var total int64
	var fi os.FileInfo
	var err error

	s := time.Now()

	// Convert PNG/JPG to JPG with quility
	if iConvert == ccconvert.Png2Jpg || iConvert == ccconvert.Jpg2Jpg {
		if iQuality < 1 || iQuality > 100 {
			useAge()
			return
		}

		if len(sSrc) == 0 || len(sDst) == 0 {
			useAge()
			return
		}

		err = ccconvert.Convert(sSrc, sDst, iQuality, nil, func(file *os.File) (image.Image, error) {
			switch iConvert {
			case ccconvert.Png2Jpg:
				return png.Decode(file)
			case ccconvert.Jpg2Jpg:
				return jpeg.Decode(file)
			default:
				return nil, nil
			}
		}, func(file *os.File, rgba *image.RGBA, options *jpeg.Options) error {
			switch iConvert {
			case ccconvert.Png2Jpg,
				ccconvert.Jpg2Jpg:
				return jpeg.Encode(file, rgba, options)
			}
			return nil
		})
		if err != nil {
			fmt.Println(err)
		}
		goto Finished
	}

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

Finished:
	cost := time.Now().Unix() - s.Unix()
	log.Printf("Total[%v].finished!...cost[%v s].err[%v]", total, cost, err)
	return
}
