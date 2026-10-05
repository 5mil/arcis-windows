package host

import (
	"crypto/sha1"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

// Draw a PNG on this PC. Arcis does not call a remote picture model.

func (h *Host) drawImage(prompt string) (string, error) {
	dir := filepath.Join(h.root, "media")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	sum := sha1.Sum([]byte(prompt))
	name := fmt.Sprintf("%x.png", sum[:6])
	path := filepath.Join(dir, name)
	img := image.NewRGBA(image.Rect(0, 0, 640, 360))
	bg := color.RGBA{17, 24, 39, 255}
	gold := color.RGBA{245, 158, 11, 255}
	green := color.RGBA{34, 197, 94, 255}
	for y := 0; y < 360; y++ {
		for x := 0; x < 640; x++ {
			img.Set(x, y, bg)
		}
	}
	fill(img, 70, 80, 180, 180, gold)
	fill(img, 390, 80, 180, 180, green)
	label(img, 24, 300, prompt, color.RGBA{248, 250, 252, 255})
	label(img, 24, 328, "arcis media", color.RGBA{148, 163, 184, 255})
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		return "", err
	}
	return name, nil
}

func fill(img *image.RGBA, x, y, w, h int, c color.Color) {
	for yy := y; yy < y+h && yy < img.Bounds().Dy(); yy++ {
		for xx := x; xx < x+w && xx < img.Bounds().Dx(); xx++ {
			img.Set(xx, yy, c)
		}
	}
}

func label(img *image.RGBA, x, y int, text string, c color.Color) {
	text = strings.ToUpper(text)
	for i, ch := range text {
		if i > 42 {
			break
		}
		glyph(img, x+i*14, y, byte(ch), c)
	}
}

func glyph(img *image.RGBA, x, y int, ch byte, c color.Color) {
	// 5-column bit rows for a few letters. Unknown characters get a block.
	rows := []byte{0x1f, 0x11, 0x11, 0x11, 0x1f}
	switch ch {
	case ' ':
		return
	case 'A':
		rows = []byte{0x0e, 0x11, 0x1f, 0x11, 0x11}
	case 'C':
		rows = []byte{0x0e, 0x11, 0x10, 0x11, 0x0e}
	case 'E':
		rows = []byte{0x1f, 0x10, 0x1e, 0x10, 0x1f}
	case 'G':
		rows = []byte{0x0e, 0x10, 0x17, 0x11, 0x0e}
	case 'I':
		rows = []byte{0x1f, 0x04, 0x04, 0x04, 0x1f}
	case 'K':
		rows = []byte{0x11, 0x12, 0x1c, 0x12, 0x11}
	case 'M':
		rows = []byte{0x11, 0x1b, 0x15, 0x11, 0x11}
	case 'O':
		rows = []byte{0x0e, 0x11, 0x11, 0x11, 0x0e}
	case 'R':
		rows = []byte{0x1e, 0x11, 0x1e, 0x12, 0x11}
	case 'S':
		rows = []byte{0x0f, 0x10, 0x0e, 0x01, 0x1e}
	case 'T':
		rows = []byte{0x1f, 0x04, 0x04, 0x04, 0x04}
	case 'U':
		rows = []byte{0x11, 0x11, 0x11, 0x11, 0x0e}
	}
	for row, bits := range rows {
		for col := 0; col < 5; col++ {
			if bits&(1<<uint(4-col)) == 0 {
				continue
			}
			for dy := 0; dy < 2; dy++ {
				for dx := 0; dx < 2; dx++ {
					img.Set(x+col*2+dx, y+row*2+dy, c)
				}
			}
		}
	}
}
