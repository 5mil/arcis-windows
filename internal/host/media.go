package host

import (
	"crypto/sha1"
	"fmt"
	"hash/fnv"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

// On-device picture from any prompt. No remote picture model.

func (h *Host) drawImage(prompt string) (string, error) {
	dir := filepath.Join(h.root, "media")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	sum := sha1.Sum([]byte(prompt))
	name := fmt.Sprintf("%x.png", sum[:6])
	path := filepath.Join(dir, name)
	img := image.NewRGBA(image.Rect(0, 0, 800, 480))
	hsh := fnv.New64a()
	_, _ = hsh.Write([]byte(prompt))
	seed := hsh.Sum64()
	sky := color.RGBA{uint8(40 + seed%80), uint8(80 + (seed>>8)%100), uint8(140 + (seed>>16)%80), 255}
	ground := color.RGBA{uint8(30 + (seed>>24)%40), uint8(90 + (seed>>32)%80), uint8(40 + (seed>>40)%40), 255}
	for y := 0; y < 480; y++ {
		for x := 0; x < 800; x++ {
			if y < 300 {
				img.Set(x, y, sky)
			} else {
				img.Set(x, y, ground)
			}
		}
	}
	fill(img, 80+int(seed%200), 70, 90, 90, color.RGBA{255, 220, 80, 255})
	fill(img, 460+int((seed>>8)%120), 250, 140, 160, color.RGBA{uint8(180 + seed%60), 90, 70, 255})
	label(img, 24, 420, prompt, color.RGBA{248, 250, 252, 255})
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
	b := img.Bounds()
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			if xx >= b.Min.X && yy >= b.Min.Y && xx < b.Max.X && yy < b.Max.Y {
				img.Set(xx, yy, c)
			}
		}
	}
}

func label(img *image.RGBA, x, y int, text string, c color.Color) {
	text = strings.ToUpper(text)
	col := 0
	for _, ch := range text {
		if col > 48 {
			break
		}
		glyph(img, x+col*14, y, byte(ch), c)
		col++
	}
}

func glyph(img *image.RGBA, x, y int, ch byte, c color.Color) {
	rows := []byte{0x1f, 0x11, 0x11, 0x11, 0x1f}
	switch ch {
	case ' ', '-', '_':
		return
	case 'A':
		rows = []byte{0x0e, 0x11, 0x1f, 0x11, 0x11}
	case 'B':
		rows = []byte{0x1e, 0x11, 0x1e, 0x11, 0x1e}
	case 'C':
		rows = []byte{0x0e, 0x11, 0x10, 0x11, 0x0e}
	case 'D':
		rows = []byte{0x1e, 0x11, 0x11, 0x11, 0x1e}
	case 'E':
		rows = []byte{0x1f, 0x10, 0x1e, 0x10, 0x1f}
	case 'F':
		rows = []byte{0x1f, 0x10, 0x1e, 0x10, 0x10}
	case 'G':
		rows = []byte{0x0e, 0x10, 0x17, 0x11, 0x0e}
	case 'H':
		rows = []byte{0x11, 0x11, 0x1f, 0x11, 0x11}
	case 'I':
		rows = []byte{0x1f, 0x04, 0x04, 0x04, 0x1f}
	case 'L':
		rows = []byte{0x10, 0x10, 0x10, 0x10, 0x1f}
	case 'M':
		rows = []byte{0x11, 0x1b, 0x15, 0x11, 0x11}
	case 'N':
		rows = []byte{0x11, 0x19, 0x15, 0x13, 0x11}
	case 'O':
		rows = []byte{0x0e, 0x11, 0x11, 0x11, 0x0e}
	case 'P':
		rows = []byte{0x1e, 0x11, 0x1e, 0x10, 0x10}
	case 'R':
		rows = []byte{0x1e, 0x11, 0x1e, 0x12, 0x11}
	case 'S':
		rows = []byte{0x0f, 0x10, 0x0e, 0x01, 0x1e}
	case 'T':
		rows = []byte{0x1f, 0x04, 0x04, 0x04, 0x04}
	case 'U':
		rows = []byte{0x11, 0x11, 0x11, 0x11, 0x0e}
	case 'W':
		rows = []byte{0x11, 0x11, 0x15, 0x1b, 0x11}
	case 'Y':
		rows = []byte{0x11, 0x0a, 0x04, 0x04, 0x04}
	}
	for row, bits := range rows {
		for col := 0; col < 5; col++ {
			if bits&(1<<uint(4-col)) == 0 {
				continue
			}
			img.Set(x+col*2, y+row*2, c)
			img.Set(x+col*2+1, y+row*2, c)
			img.Set(x+col*2, y+row*2+1, c)
			img.Set(x+col*2+1, y+row*2+1, c)
		}
	}
}
