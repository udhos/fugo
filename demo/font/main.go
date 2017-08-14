package main

import (
	"image"
	"image/color"
	"image/png"
	"os"

	"github.com/udhos/pixfont"
)

func main() {
	img := image.NewRGBA(image.Rect(0, 0, 150, 30))

	pixfont.DrawString(img, 10, 10, "Hello, World!", color.Black)

	f, _ := os.OpenFile("hello.png", os.O_CREATE|os.O_RDWR, 0644)
	png.Encode(f, img)
}
