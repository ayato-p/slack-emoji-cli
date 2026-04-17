package render

import (
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

// loadFont はファイルパスからフォントを読み込みます。
func loadFont(path string) (*opentype.Font, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return opentype.Parse(data)
}

// loadFace は与えられたフォントから指定サイズのフェイスを作成します。
func loadFace(f *opentype.Font, size float64) (font.Face, error) {
	return opentype.NewFace(f, &opentype.FaceOptions{
		Size: size,
		DPI:  72,
	})
}
