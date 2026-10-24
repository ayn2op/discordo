package chat

import (
	"bytes"
	"compress/zlib"
	"image"
	"io"
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

func TestNewPooledCompressor(t *testing.T) {
	want := []byte("pixels")
	// The second round reuses the pooled writer.
	for range 2 {
		var buf bytes.Buffer
		compressor, err := newPooledCompressor(&buf)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := compressor.Write(want); err != nil {
			t.Fatal(err)
		}
		if err := compressor.Close(); err != nil {
			t.Fatal(err)
		}

		reader, err := zlib.NewReader(&buf)
		if err != nil {
			t.Fatal(err)
		}
		got, err := io.ReadAll(reader)
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("got %q, %v, want %q", got, err, want)
		}
	}
}
