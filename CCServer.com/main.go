package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"sort"
	"time"

	"CCServer.com/cccompress"
	"CCServer.com/ccconvert"
)

var (
	bConvert bool
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
	flag.BoolVar(&bConvert, "convert", false, "Convert PNG/JPG/JPEG to JPG with custom image quality  default:false")
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
	log.Printf("%s", cmdStr)

	flag.PrintDefaults()
}

func convertTo8Bit(img image.Image) *image.Paletted {
	bounds := img.Bounds()
	// 创建一个包含透明色的256色调色板
	plt := make(color.Palette, 0, 256)
	plt = append(plt, color.Transparent) // 首先添加透明色

	// 添加其他颜色，确保总数不超过256
	for _, c := range palette.Plan9 {
		if len(plt) >= 256 {
			break
		}
		plt = append(plt, c)
	}

	paletted := image.NewPaletted(bounds, plt)
	// draw.Draw(paletted, bounds, img, bounds.Min, draw.Src)
	draw.FloydSteinberg.Draw(paletted, bounds, img, image.Point{})
	return paletted
}

func convertTo8Bit2(img image.Image) *image.Paletted {
	bounds := img.Bounds()

	// 统计图像中的颜色分布
	colorCount := make(map[color.Color]int)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.At(x, y)
			colorCount[c]++
		}
	}

	// 将颜色按使用频率排序
	type colorFrequency struct {
		color     color.Color
		frequency int
	}
	var sortedColors []colorFrequency
	for c, freq := range colorCount {
		sortedColors = append(sortedColors, colorFrequency{c, freq})
	}

	// 按频率从高到低排序
	sort.Slice(sortedColors, func(i, j int) bool {
		return sortedColors[i].frequency > sortedColors[j].frequency
	})

	// 创建调色板，最多包含 256 种颜色
	palette := make(color.Palette, 0, 256)
	palette = append(palette, color.Transparent) // 确保透明色优先
	for _, cf := range sortedColors {
		if len(palette) >= 256 {
			break
		}
		palette = append(palette, cf.color)
	}

	// 创建 Paletted 图像
	paletted := image.NewPaletted(bounds, palette)
	draw.FloydSteinberg.Draw(paletted, bounds, img, image.Point{})

	return paletted
}

func convertTo8Bit3(img image.Image) *image.Paletted {
	// 使用中值切割算法生成调色板
	palette := ccconvert.MedianCut(img, 256)

	// 创建 Paletted 图像
	bounds := img.Bounds()
	paletted := image.NewPaletted(bounds, palette)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			paletted.Set(x, y, img.At(x, y))
		}
	}
	draw.FloydSteinberg.Draw(paletted, bounds, img, image.Point{})
	return paletted
}

func convertTo8Bit4(img image.Image) *image.Paletted {
	bounds := img.Bounds()

	// 1. 保留最鲜艳的颜色
	colorMap := make(map[color.Color]int)
	maxR, maxG, maxB := 0, 0, 0

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.At(x, y)
			r, g, b, _ := c.RGBA()

			// 记录最高饱和度的颜色
			if r > uint32(maxR) {
				maxR = int(r)
			}
			if g > uint32(maxG) {
				maxG = int(g)
			}
			if b > uint32(maxB) {
				maxB = int(b)
			}

			colorMap[c]++
		}
	}

	// 2. 创建包含鲜艳色彩的调色板
	plt := make(color.Palette, 0, 256)
	plt = append(plt, color.Transparent)

	// 3. 添加饱和度最高的颜色
	plt = append(plt, color.RGBA{uint8(maxR >> 8), uint8(maxG >> 8), uint8(maxB >> 8), 255})

	// 4. 使用改进的中值切割填充剩余调色板
	remainingColors := ccconvert.MedianCut(img, 254) // 预留了2个位置
	plt = append(plt, remainingColors...)

	// 5. 使用误差扩散保持细节
	paletted := image.NewPaletted(bounds, plt)
	draw.FloydSteinberg.Draw(paletted, bounds, img, image.Point{})

	return paletted
}

func modifyDPI(pngData []byte, dpi int) ([]byte, error) {
	// 将 DPI 转换为每米像素数 (1 英寸 = 0.0254 米)
	ppm := uint32(float64(dpi) / 0.0254)

	// 创建 pHYs 块数据
	physData := make([]byte, 9)
	binary.BigEndian.PutUint32(physData[0:4], ppm) // 水平像素密度
	binary.BigEndian.PutUint32(physData[4:8], ppm) // 垂直像素密度
	physData[8] = 1                                // 单位：每米像素数

	// 计算 CRC 校验值
	crc := crc32.Checksum(append([]byte("pHYs"), physData...), crc32.MakeTable(crc32.IEEE))

	// 构造 pHYs 块
	var physChunk bytes.Buffer
	binary.Write(&physChunk, binary.BigEndian, uint32(len(physData))) // 块长度
	physChunk.WriteString("pHYs")                                     // 块类型
	physChunk.Write(physData)                                         // 块数据
	binary.Write(&physChunk, binary.BigEndian, crc)                   // CRC 校验值

	// 找到第一个 IDAT 块的位置
	idatIndex := bytes.Index(pngData, []byte("IDAT")) - 4
	if idatIndex < 0 {
		return nil, fmt.Errorf("未找到 IDAT 块")
	}

	// 将 pHYs 块插入到 IDAT 块之前
	newData := append(pngData[:idatIndex], append(physChunk.Bytes(), pngData[idatIndex:]...)...)
	return newData, nil
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

	// Convert PNG/JPG/JPEG to JPG with custom image quality
	if bConvert {
		if iQuality < 1 || iQuality > 100 {
			useAge()
			return
		}

		if len(sSrc) == 0 || len(sDst) == 0 {
			useAge()
			return
		}

		ext := ""

		err = ccconvert.Convert(sSrc, sDst, nil, func(file *os.File, _ext string) (image.Image, error) {
			ext = _ext
			switch ext {
			case "image/png":
				return png.Decode(file)
			case "image/jpeg":
				return jpeg.Decode(file)
			default:
				return nil, nil
			}
		}, func(file *os.File, rgba *image.RGBA, options *jpeg.Options) error {
			switch ext {
			case "image/png":
				// 获取图像的基本信息
				bounds := rgba.Bounds()
				width, height := bounds.Dx(), bounds.Dy()
				colorModel := rgba.ColorModel()

				fmt.Printf("宽度: %d\n", width)
				fmt.Printf("高度: %d\n", height)
				fmt.Printf("颜色模型: %v\n", colorModel)
				fmt.Printf("颜色数: %d\n", len(palette.Plan9))

				// 转8bit位深
				palettedImg := convertTo8Bit(rgba)

				enc := &png.Encoder{
					CompressionLevel: png.BestCompression,
				}

				// return enc.Encode(file, palettedImg)

				var buf bytes.Buffer
				// err = png.Encode(&buf, palettedImg)
				err = enc.Encode(&buf, palettedImg)
				if err != nil {
					return err
				}

				modifiedData, err := modifyDPI(buf.Bytes(), 72)
				if err != nil {
					return err
				}
				_, err = file.Write(modifiedData)
				return err

				// // 转换为8位调色板图像
				// bounds := rgba.Bounds()

				// // 创建一个包含透明色的256色调色板
				// plt := make(color.Palette, 0, 256)
				// plt = append(plt, color.Transparent) // 首先添加透明色

				// // 添加其他颜色，确保总数不超过256
				// for _, c := range palette.Plan9 {
				// 	if len(plt) >= 256 {
				// 		break
				// 	}
				// 	plt = append(plt, c)
				// }

				// paletted := image.NewPaletted(bounds, plt) // Plan9 是一个256色调色板
				// draw.FloydSteinberg.Draw(paletted, bounds, rgba, image.Point{})

				// enc := &png.Encoder{
				// 	CompressionLevel: png.BestCompression,
				// }
				// return enc.Encode(file, paletted)
			case "image/jpeg":
				options.Quality = iQuality
				return jpeg.Encode(file, rgba, options)
			}
			return nil
		})
		if err != nil {
			fmt.Println(err)
		}
		if ext == "image/png" {
			ccconvert.RemoveMetaData(sDst)
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
}
