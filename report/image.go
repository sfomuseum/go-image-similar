package report

import (
	"fmt"
	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
	"image"
	"image/color"
)

// CombineImagesInput is a struct containing image data to be passed to the `CombineImages` method.
type CombineImagesInput struct {
	// The underlying `image.Image` instance being combined.	
	Image      image.Image
	// The "score" of the image being combined relative to another image (typically the first in a list of the `CombineImagesInput` instances).
	Similarity float32
}

// CombineImages() sorts all the images defined by 'images' according to their "similarity" score in ascending
// order and then combines them in to a montage with (1) row and (n) columns. Each image is scaled to a maximum
// dimension of 250 pixels.
func CombineImages(input []*CombineImagesInput) (image.Image, error) {

	// https://github.com/ashleymcnamara/artwork/blob/master/collage.go

	cols := len(input)
	cell := 250

	rows := (cols + cols - 1) / cols
	dst := image.NewRGBA(image.Rect(0, 0, cell*cols, cell*rows))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.RGBA{0xFF, 0xFF, 0xFF, 0xFF}), image.Point{}, draw.Src)

	for i, m := range input {

		im := m.Image

		col := i % cols
		row := i / cols

		sz := im.Bounds().Size()
		dz := sz
		if sz.X > sz.Y {
			dz.X = cell
			dz.Y = cell * sz.Y / sz.X
		} else {
			dz.Y = cell
			dz.X = cell * sz.X / sz.Y
		}

		z := image.Point{cell * col, cell * row}
		r := image.Rectangle{
			Min: z,
			Max: z.Add(dz),
		}
		r = r.Add(image.Point{cell / 2, cell / 2}).
			Sub(image.Point{dz.X / 2, dz.Y / 2})

		draw.CatmullRom.Scale(dst, r, im, im.Bounds(), draw.Over, nil)

		// fmt.Printf("Add label '%s' at %d, %d\n", fmt.Sprintf("%f", m.Similarity), r.Min.X, r.Min.Y)
		addLabel(dst, r.Min.X+5, 15, fmt.Sprintf("%f", m.Similarity))
	}

	return dst, nil
}

func addLabel(img *image.RGBA, x, y int, label string) {
	col := color.RGBA{255, 255, 255, 255}
	point := fixed.Point26_6{fixed.I(x), fixed.I(y)}

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: basicfont.Face7x13,
		Dot:  point,
	}
	d.DrawString(label)
}
