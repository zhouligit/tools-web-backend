package imageproc

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	"image/jpeg"
	"image/png"

	_ "golang.org/x/image/bmp"
	"io"
	"strings"

	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
)

type Format string

const (
	FormatJPEG Format = "jpg"
	FormatPNG  Format = "png"
	FormatWebP Format = "webp"
)

type Result struct {
	Data         []byte
	MIME         string
	Ext          string
	OriginalSize int
	OutputSize   int
	Width        int
	Height       int
}

type Processor struct{}

func NewProcessor() *Processor {
	return &Processor{}
}

func ParseFormat(raw string) (Format, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "jpg", "jpeg":
		return FormatJPEG, nil
	case "png":
		return FormatPNG, nil
	case "webp":
		return FormatWebP, nil
	default:
		return "", fmt.Errorf("unsupported format: %s", raw)
	}
}

func (p *Processor) Convert(input []byte, filename string, target Format, quality int) (*Result, error) {
	img, srcFormat, err := p.decode(input)
	if err != nil {
		return nil, err
	}
	if quality <= 0 || quality > 100 {
		quality = 85
	}
	data, mime, ext, err := p.encode(img, target, quality)
	if err != nil {
		return nil, err
	}
	_ = srcFormat
	bounds := img.Bounds()
	return &Result{
		Data:         data,
		MIME:         mime,
		Ext:          ext,
		OriginalSize: len(input),
		OutputSize:   len(data),
		Width:        bounds.Dx(),
		Height:       bounds.Dy(),
	}, nil
}

func (p *Processor) Compress(input []byte, quality int, maxEdge int, outputFormat string) (*Result, error) {
	img, srcFormat, err := p.decode(input)
	if err != nil {
		return nil, err
	}
	if quality <= 0 || quality > 100 {
		quality = 85
	}
	if maxEdge > 0 {
		img = resizeMaxEdge(img, maxEdge)
	}

	target, err := p.resolveCompressFormat(srcFormat, outputFormat)
	if err != nil {
		return nil, err
	}
	data, mime, ext, err := p.encode(img, target, quality)
	if err != nil {
		return nil, err
	}
	bounds := img.Bounds()
	return &Result{
		Data:         data,
		MIME:         mime,
		Ext:          ext,
		OriginalSize: len(input),
		OutputSize:   len(data),
		Width:        bounds.Dx(),
		Height:       bounds.Dy(),
	}, nil
}

func (p *Processor) resolveCompressFormat(srcFormat string, outputFormat string) (Format, error) {
	switch strings.ToLower(strings.TrimSpace(outputFormat)) {
	case "", "keep", "original":
		return formatFromName(srcFormat)
	case "webp":
		return FormatWebP, nil
	default:
		return ParseFormat(outputFormat)
	}
}

func formatFromName(name string) (Format, error) {
	switch strings.ToLower(name) {
	case "jpeg", "jpg":
		return FormatJPEG, nil
	case "png":
		return FormatPNG, nil
	case "webp":
		return FormatWebP, nil
	case "gif":
		return FormatPNG, nil
	case "bmp":
		return FormatPNG, nil
	default:
		return FormatJPEG, nil
	}
}

func (p *Processor) decode(input []byte) (image.Image, string, error) {
	img, format, err := image.Decode(bytes.NewReader(input))
	if err == nil {
		return img, format, nil
	}
	// Fallback for some WebP inputs.
	wimg, werr := webp.Decode(bytes.NewReader(input))
	if werr == nil {
		return wimg, "webp", nil
	}
	return nil, "", fmt.Errorf("unsupported or corrupt image")
}

func (p *Processor) encode(img image.Image, format Format, quality int) ([]byte, string, string, error) {
	switch format {
	case FormatJPEG:
		img = flattenAlpha(img)
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
			return nil, "", "", err
		}
		return buf.Bytes(), "image/jpeg", "jpg", nil
	case FormatPNG:
		var buf bytes.Buffer
		enc := png.Encoder{CompressionLevel: png.BestCompression}
		if err := enc.Encode(&buf, img); err != nil {
			return nil, "", "", err
		}
		return buf.Bytes(), "image/png", "png", nil
	case FormatWebP:
		var buf bytes.Buffer
		if err := webp.Encode(&buf, img, &webp.Options{Quality: float32(quality)}); err != nil {
			return nil, "", "", err
		}
		return buf.Bytes(), "image/webp", "webp", nil
	default:
		return nil, "", "", fmt.Errorf("unsupported output format")
	}
}

func flattenAlpha(src image.Image) image.Image {
	b := src.Bounds()
	dst := imaging.New(b.Dx(), b.Dy(), color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	return imaging.Paste(dst, src, b.Min)
}

func resizeMaxEdge(img image.Image, maxEdge int) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxEdge && h <= maxEdge {
		return img
	}
	if w >= h {
		return imaging.Resize(img, maxEdge, 0, imaging.Lanczos)
	}
	return imaging.Resize(img, 0, maxEdge, imaging.Lanczos)
}

func OutputFilename(original string, ext string) string {
	base := strings.TrimSuffix(original, extWithDot(original))
	if base == "" {
		base = "image"
	}
	return base + "." + ext
}

func extWithDot(filename string) string {
	idx := strings.LastIndex(filename, ".")
	if idx <= 0 {
		return ""
	}
	return filename[idx:]
}

func ReadImageFile(r io.Reader, maxBytes int64) ([]byte, error) {
	if maxBytes <= 0 {
		return io.ReadAll(r)
	}
	limited := io.LimitReader(r, maxBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("file too large, max %d MB", maxBytes/(1024*1024))
	}
	return data, nil
}
