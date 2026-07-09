package imageproc

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func makeTestPNG(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func TestConvertPNGToJPEG(t *testing.T) {
	p := NewProcessor()
	input := makeTestPNG(100, 80)
	result, err := p.Convert(input, "test.png", FormatJPEG, 85)
	if err != nil {
		t.Fatal(err)
	}
	if result.MIME != "image/jpeg" {
		t.Fatalf("expected jpeg, got %s", result.MIME)
	}
	if result.OutputSize <= 0 {
		t.Fatalf("unexpected output size: %d", result.OutputSize)
	}
	if result.Width != 100 || result.Height != 80 {
		t.Fatalf("unexpected dimensions: %dx%d", result.Width, result.Height)
	}
}

func TestCompressWithResize(t *testing.T) {
	p := NewProcessor()
	input := makeTestPNG(800, 600)
	result, err := p.Compress(input, 80, 400, "keep")
	if err != nil {
		t.Fatal(err)
	}
	if result.Width > 400 || result.Height > 400 {
		t.Fatalf("expected max edge 400, got %dx%d", result.Width, result.Height)
	}
	if result.OutputSize >= result.OriginalSize {
		t.Fatalf("expected smaller output, before=%d after=%d", result.OriginalSize, result.OutputSize)
	}
}

func TestConvertToWebP(t *testing.T) {
	p := NewProcessor()
	input := makeTestPNG(64, 64)
	result, err := p.Convert(input, "a.png", FormatWebP, 85)
	if err != nil {
		t.Fatal(err)
	}
	if result.Ext != "webp" || result.MIME != "image/webp" {
		t.Fatalf("unexpected type: %s %s", result.MIME, result.Ext)
	}
}

func TestReadImageFileLimit(t *testing.T) {
	data := makeTestPNG(10, 10)
	_, err := ReadImageFile(bytes.NewReader(data), 1)
	if err == nil {
		t.Fatal("expected size limit error")
	}
}

func TestOutputFilename(t *testing.T) {
	if got := OutputFilename("photo.PNG", "jpg"); got != "photo.jpg" {
		t.Fatalf("got %s", got)
	}
}
