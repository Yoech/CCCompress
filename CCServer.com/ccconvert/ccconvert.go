package ccconvert

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"net/http"
	"os"
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
