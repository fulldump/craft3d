package main

import (
	"image"
	"image/color"
	"math"
)

// DrawLine draws a line on the image from (x1, y1) to (x2, y2) using Bresenham's algorithm
func DrawLine(img *image.RGBA, x1, y1, x2, y2 int, col color.RGBA) {
	dx := float64(x2 - x1)
	dy := float64(y2 - y1)
	steps := math.Max(math.Abs(dx), math.Abs(dy))

	xInc := dx / steps
	yInc := dy / steps

	x := float64(x1)
	y := float64(y1)

	for i := 0; i <= int(steps); i++ {
		ix := int(x)
		iy := int(y)
		if ix >= 0 && ix < img.Bounds().Dx() && iy >= 0 && iy < img.Bounds().Dy() {
			offset := img.PixOffset(ix, iy)
			img.Pix[offset] = col.R
			img.Pix[offset+1] = col.G
			img.Pix[offset+2] = col.B
			img.Pix[offset+3] = col.A
		}
		x += xInc
		y += yInc
	}
}
