package main

import (
	"image"
	_ "image/png"
	"os"
	"strings"
	"testing"
)

func loadGray(t *testing.T, p string) ([]float64, int, int) {
	f, err := os.Open(p)
	if err != nil {
		t.Skip(err)
	}
	defer f.Close()
	im, _, err := image.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	b := im.Bounds()
	w, h := b.Dx(), b.Dy()
	g := make([]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, gg, bb, _ := im.At(b.Min.X+x, b.Min.Y+y).RGBA()
			g[y*w+x] = (0.299*float64(r) + 0.587*float64(gg) + 0.114*float64(bb)) / 257
		}
	}
	return g, w, h
}

func TestDetectShots(t *testing.T) {
	dataDir = "/tmp/hrd"
	if loadGameData() != nil {
		t.Skip()
	}
	d := getGD()
	for _, p := range []string{"/mnt/user-data/uploads/75a7addb-image.png", "/mnt/user-data/uploads/ab2b1587-image.png"} {
		g, w, h := loadGray(t, p)
		detDebug = func(P, bx, by float64, nx, ny int, pres []bool) {
			var sb strings.Builder
			for j := 0; j < ny; j++ {
				for i := 0; i < nx; i++ {
					if pres[j*nx+i] {
						sb.WriteByte('#')
					} else {
						sb.WriteByte('.')
					}
				}
				sb.WriteByte('\n')
			}
			t.Logf("P=%.1f bx=%.0f by=%.0f\n%s", P, bx, by, sb.String())
			detDebug = nil
		}

		for k, b := range d.Boards {
			if !strings.HasPrefix(k, "Paragon_Barb") {
				continue
			}
			raw := make([]byte, len(b.Cells))
			for i, c := range b.Cells {
				raw[i] = '0' + c
			}
			x := refineDetect(g, w, h, detectBoard(g, w, h, string(raw), b.Width, 14, 140), string(raw), b.Width)
			if x.OK {
				t.Logf("  centro della colonna 10, riga 14: %.0f, %.0f", x.X+10.5*x.Cell, x.Y+14.5*x.Cell)
			}
			if x.OK || x.Score > 0.3 {
				t.Logf("%s %s %s: %+v", p[len(p)-14:], k, b.En, x)
			}
		}
	}
}
