package ccconvert

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"io"
	"net/http"
	"os"
	"sort"
)

const (
	UnknownConvertMode = 0
	Png2Jpg            = 1
	Jpg2Jpg            = 2
)

func readRaw(src string, decode func(file *os.File, ext string) (image.Image, error)) (image.Image, error) {
	f, err := os.Open(src)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	defer f.Close()

	buff := make([]byte, 512)
	_, err = f.Read(buff)
	if err != nil {
		return nil, err
	}

	// seek to begin
	// Cool.Cat
	f.Seek(0, 0)

	var img image.Image
	ext := http.DetectContentType(buff)
	img, err = decode(f, ext)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return img, nil
}

func RemoveMetaData(dst string) error {
	file, err := os.Open(dst)
	if err != nil {
		return fmt.Errorf("无法打开文件: %w", err)
	}

	// 跳过 PNG 文件签名 (8 字节)
	header := make([]byte, 8)
	_, err = file.Read(header)
	if err != nil {
		return fmt.Errorf("读取文件头失败: %w", err)
	}

	// 检查文件签名是否为有效的 PNG 文件
	if !bytes.Equal(header, []byte{137, 80, 78, 71, 13, 10, 26, 10}) {
		return fmt.Errorf("这不是一个有效的 PNG 文件")
	}

	var output bytes.Buffer
	output.Write(header) // 写入 PNG 文件签名

	for {
		// 读取数据块长度 (4 字节)
		var length uint32
		err := binary.Read(file, binary.BigEndian, &length)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("读取数据块长度失败: %w", err)
		}

		// 读取数据块类型 (4 字节)
		var chunkType [4]byte
		_, err = file.Read(chunkType[:])
		if err != nil {
			return fmt.Errorf("读取数据块类型失败: %w", err)
		}

		// 读取数据块内容 (length 字节)
		data := make([]byte, length)
		_, err = file.Read(data)
		if err != nil {
			return fmt.Errorf("读取数据块内容失败: %w", err)
		}

		// 读取数据块 CRC (4 字节)
		var crc uint32
		err = binary.Read(file, binary.BigEndian, &crc)
		if err != nil {
			return fmt.Errorf("读取数据块 CRC 失败: %w", err)
		}

		// 只保留必要的数据块 (IHDR, PLTE, IDAT, IEND)
		switch string(chunkType[:]) {
		case "IHDR", "PLTE", "IDAT", "IEND", "pHYs":
			// 写入数据块到输出文件
			binary.Write(&output, binary.BigEndian, length)
			output.Write(chunkType[:])
			output.Write(data)
			binary.Write(&output, binary.BigEndian, crc)
		default:
			fmt.Printf("移除不必要的数据块: %s\n", string(chunkType[:]))
		}
	}

	file.Close()

	// 将结果写入输出文件
	outFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("无法创建输出文件: %w", err)
	}
	defer outFile.Close()

	_, err = output.WriteTo(outFile)
	if err != nil {
		return fmt.Errorf("写入输出文件失败: %w", err)
	}

	return nil
}

func Convert(src, dst string, bgColor color.Color, decode func(file *os.File, ext string) (image.Image, error), encode func(file *os.File, rgba *image.RGBA, options *jpeg.Options) error) error {
	img, err := readRaw(src, decode)
	if img == nil {
		return err
	}
	var out *os.File
	out, err = os.Create(dst)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer out.Close()

	jpg := image.NewRGBA(image.Rect(0, 0, img.Bounds().Max.X, img.Bounds().Max.Y))

	if bgColor == nil {
		// Draw image to background
		draw.Draw(jpg, jpg.Bounds(), img, img.Bounds().Min, draw.Src)
	} else {
		// Draw background using custom colors
		draw.Draw(jpg, jpg.Bounds(), &image.Uniform{C: bgColor}, image.Point{}, draw.Src)

		// Draw image to new background
		draw.Draw(jpg, jpg.Bounds(), img, img.Bounds().Min, draw.Over)
	}

	// Encode to dest image format
	return encode(out, jpg, &jpeg.Options{Quality: 80})
}



// ColorBox 表示一个颜色空间的范围
type ColorBox struct {
	Colors []color.Color // 包含的颜色
	MinRGB [3]int        // RGB 最小值
	MaxRGB [3]int        // RGB 最大值
}

// MedianCut 使用中值切割算法生成调色板
func MedianCut(img image.Image, numColors int) color.Palette {
	bounds := img.Bounds()

	// 收集图像中的所有颜色
	var colors []color.RGBA
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			colors = append(colors, color.RGBA{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(b >> 8),
				A: 255,
			})
		}
	}

	// 初始化颜色空间
	initialBox := ColorBox{
		Colors: toColorSlice(colors),
		MinRGB: [3]int{0, 0, 0},
		MaxRGB: [3]int{255, 255, 255},
	}
	boxes := []*ColorBox{&initialBox}

	// 中值切割过程
	for len(boxes) < numColors {
		// 找到跨度最大的颜色空间
		maxBoxIndex := 0
		maxRange := 0
		for i, box := range boxes {
			rangeSize := box.MaxRGB[0] - box.MinRGB[0]
			if box.MaxRGB[1]-box.MinRGB[1] > rangeSize {
				rangeSize = box.MaxRGB[1] - box.MinRGB[1]
			}
			if box.MaxRGB[2]-box.MinRGB[2] > rangeSize {
				rangeSize = box.MaxRGB[2] - box.MinRGB[2]
			}
			if rangeSize > maxRange {
				maxRange = rangeSize
				maxBoxIndex = i
			}
		}

		// 对选定的颜色空间进行分割
		box := boxes[maxBoxIndex]
		boxes = append(boxes[:maxBoxIndex], boxes[maxBoxIndex+1:]...)
		boxes = append(boxes, splitBox(box)...)
	}

	// 生成调色板
	palette := make(color.Palette, len(boxes))
	for i, box := range boxes {
		palette[i] = averageColor(box.Colors)
	}

	return palette
}

// splitBox 将一个颜色空间按中值分割成两个子空间
func splitBox(box *ColorBox) []*ColorBox {
	// 找到跨度最大的维度
	dim := 0
	maxRange := box.MaxRGB[0] - box.MinRGB[0]
	if box.MaxRGB[1]-box.MinRGB[1] > maxRange {
		dim = 1
		maxRange = box.MaxRGB[1] - box.MinRGB[1]
	}
	if box.MaxRGB[2]-box.MinRGB[2] > maxRange {
		dim = 2
	}

	// 按该维度排序颜色
	sort.Slice(box.Colors, func(i, j int) bool {
		r1, g1, b1, _ := box.Colors[i].RGBA()
		r2, g2, b2, _ := box.Colors[j].RGBA()
		switch dim {
		case 0:
			return r1 < r2
		case 1:
			return g1 < g2
		case 2:
			return b1 < b2
		default:
			return false
		}
	})

	// 按中值分割
	mid := len(box.Colors) / 2
	leftColors := box.Colors[:mid]
	rightColors := box.Colors[mid:]

	// 创建两个子空间
	leftBox := &ColorBox{
		Colors: leftColors,
		MinRGB: box.MinRGB,
		MaxRGB: box.MaxRGB,
	}
	rightBox := &ColorBox{
		Colors: rightColors,
		MinRGB: box.MinRGB,
		MaxRGB: box.MaxRGB,
	}

	// 更新子空间的最大最小值
	updateMinMax(leftBox)
	updateMinMax(rightBox)

	return []*ColorBox{leftBox, rightBox}
}

// updateMinMax 更新颜色空间的最大最小值
func updateMinMax(box *ColorBox) {
	for _, c := range box.Colors {
		r, g, b, _ := c.RGBA()
		if int(r>>8) < box.MinRGB[0] {
			box.MinRGB[0] = int(r >> 8)
		}
		if int(g>>8) < box.MinRGB[1] {
			box.MinRGB[1] = int(g >> 8)
		}
		if int(b>>8) < box.MinRGB[2] {
			box.MinRGB[2] = int(b >> 8)
		}
		if int(r>>8) > box.MaxRGB[0] {
			box.MaxRGB[0] = int(r >> 8)
		}
		if int(g>>8) > box.MaxRGB[1] {
			box.MaxRGB[1] = int(g >> 8)
		}
		if int(b>>8) > box.MaxRGB[2] {
			box.MaxRGB[2] = int(b >> 8)
		}
	}
}

// averageColor 计算一组颜色的平均值
func averageColor(colors []color.Color) color.Color {
	var rSum, gSum, bSum int
	for _, c := range colors {
		r, g, b, _ := c.RGBA()
		rSum += int(r >> 8)
		gSum += int(g >> 8)
		bSum += int(b >> 8)
	}
	n := len(colors)
	return color.RGBA{
		R: uint8(rSum / n),
		G: uint8(gSum / n),
		B: uint8(bSum / n),
		A: 255,
	}
}

// toColorSlice 将 []color.RGBA 转换为 []color.Color
func toColorSlice(colors []color.RGBA) []color.Color {
	result := make([]color.Color, len(colors))
	for i, c := range colors {
		result[i] = c
	}
	return result
}
