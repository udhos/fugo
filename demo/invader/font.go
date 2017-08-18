// +build darwin linux windows

package main

import (
	//"fmt"
	"bytes"
	"log"
	//"os"

	"image"
	"image/color"
	//"image/png"

	//"golang.org/x/mobile/asset"
	"golang.org/x/mobile/gl"

	"github.com/udhos/pixfont"
)

type fontAtlas struct {
	glc gl.Context
	tex gl.Texture
}

func newAtlas(glc gl.Context, c color.Color) (*fontAtlas, error) {
	a := &fontAtlas{}

	first := 32
	pastend := 127
	size := pastend - first
	b := make([]byte, 0, size)
	buf := bytes.NewBuffer(b)

	for i := first; i < pastend; i++ {
		err := buf.WriteByte(byte(i))
		if err != nil {
			log.Printf("newAtlas: %d %v", i, err)
		}
	}

	str := buf.String()

	log.Printf("newAtlas: chars=%d str=%d: [%v]", size, len(str), str)

	width := pixfont.MeasureString(str)
	log.Printf("newAtlas: width=%d", width)

	img := image.NewRGBA(image.Rect(0, 0, width, 8))

	bo := img.Bounds()
	fontWidth := width / size
	fontHeight := bo.Max.Y
	log.Printf("newAtlas: atlas=%dx%d => font=%dx%d", bo.Max.X, bo.Max.Y, fontWidth, fontHeight)

	//c := color.NRGBA{128,230,128,255}
	w := pixfont.DrawString(img, 0, 0, str, c)

	log.Printf("newAtlas: drawn %d pixels", w)

	/*
		f, errWr := os.OpenFile("atlas.png", os.O_CREATE|os.O_RDWR, 0644)
		if errWr != nil {
			log.Printf("IO: %v", errWr)
		}
		png.Encode(f, img)
		//f.Flush()
		f.Close()
	*/

	a.glc = glc
	a.tex = glc.CreateTexture()

	return a, nil
}

func (a *fontAtlas) delete() {
	a.glc.DeleteTexture(a.tex)
}
