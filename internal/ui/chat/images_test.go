package chat

import (
	"image"
	"testing"
)

func TestPreviewCols(t *testing.T) {
	for _, tt := range []struct {
		name                  string
		width, height         uint
		cellWidth, cellHeight uint
		want                  int
	}{
		{"landscape", 1600, 900, 1, 2, 40},
		{"portrait", 900, 1600, 1, 2, 22},
		{"small", 10, 10, 1, 2, 10},
		{"very tall", 1, 10000, 1, 2, 1},
		{"own width in cells", 200, 100, 10, 20, 20},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := previewCols(tt.width, tt.height, tt.cellWidth, tt.cellHeight); got != tt.want {
				t.Fatalf("previewCols() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestSquare(t *testing.T) {
	t.Run("square", func(t *testing.T) {
		img := image.NewAlpha(image.Rect(0, 0, 20, 20))
		if got := square(img); got != image.Image(img) {
			t.Fatal("square image was copied")
		}
	})
	t.Run("tall", func(t *testing.T) {
		img := image.NewAlpha(image.Rect(0, 0, 10, 30))
		img.Pix[0] = 0xff
		got := square(img)
		if got.Bounds() != image.Rect(0, 0, 30, 30) {
			t.Fatalf("bounds = %v, want 30 by 30", got.Bounds())
		}
		if _, _, _, a := got.At(10, 0).RGBA(); a == 0 {
			t.Fatal("image is not centered")
		}
	})
}
