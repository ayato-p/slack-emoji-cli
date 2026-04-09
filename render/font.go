package render

import (
	_ "embed"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

//go:embed assets/NotoSansJP-Regular.ttf
var fontBytes []byte

var parsedFont *opentype.Font

func init() {
	f, err := opentype.Parse(fontBytes)
	if err != nil {
		panic("render: failed to parse embedded font: " + err.Error())
	}
	parsedFont = f
}

func loadFace(size float64) (font.Face, error) {
	return opentype.NewFace(parsedFont, &opentype.FaceOptions{
		Size: size,
		DPI:  72,
	})
}
