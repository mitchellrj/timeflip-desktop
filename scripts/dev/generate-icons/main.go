package main

import (
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"
)

type point struct {
	x float64
	y float64
}

var (
	nav      = color.RGBA{0x13, 0x2b, 0x35, 0xff}
	mint     = color.RGBA{0x7f, 0xd2, 0xa4, 0xff}
	sand     = color.RGBA{0xf5, 0xd0, 0x66, 0xff}
	template = color.RGBA{0x00, 0x00, 0x00, 0xff}
)

func main() {
	writePNG("build/assets/appicon.png", renderAppIcon(1024))
	writePNG("internal/app/icon_assets/appicon.png", renderAppIcon(512))
	writePNG("internal/app/icon_assets/tray-plain.png", renderTrayIcon(64, trayPlain))
	writePNG("internal/app/icon_assets/tray-running.png", renderTrayIcon(64, trayRunning))
	writePNG("internal/app/icon_assets/tray-paused.png", renderTrayIcon(64, trayPaused))
}

func renderAppIcon(size int) image.Image {
	scale := 3
	img := image.NewRGBA(image.Rect(0, 0, size*scale, size*scale))
	s := float64(size * scale)
	drawRoundedRect(img, 0.08*s, 0.08*s, 0.84*s, 0.84*s, 0.22*s, nav)
	drawRoundedRect(img, 0.10*s, 0.10*s, 0.80*s, 0.80*s, 0.20*s, color.RGBA{0x1b, 0x3a, 0x45, 0xff})
	drawHourglass(img, 0.27*s, 0.20*s, 0.46*s, 0.60*s, 0.055*s, mint, sand)
	return downsample(img, scale)
}

type trayState int

const (
	trayPlain trayState = iota
	trayRunning
	trayPaused
)

func renderTrayIcon(size int, state trayState) image.Image {
	scale := 4
	img := image.NewRGBA(image.Rect(0, 0, size*scale, size*scale))
	s := float64(size * scale)
	if state == trayPlain {
		drawHourglass(img, 0.25*s, 0.17*s, 0.50*s, 0.62*s, 0.045*s, template, color.RGBA{})
		return downsample(img, scale)
	}
	drawHourglass(img, 0.17*s, 0.17*s, 0.42*s, 0.62*s, 0.045*s, template, color.RGBA{})
	if state == trayPaused {
		drawRoundedRect(img, 0.68*s, 0.60*s, 0.055*s, 0.25*s, 0.018*s, template)
		drawRoundedRect(img, 0.79*s, 0.60*s, 0.055*s, 0.25*s, 0.018*s, template)
	} else {
		drawPolygon(img, []point{
			{0.68 * s, 0.57 * s},
			{0.68 * s, 0.86 * s},
			{0.89 * s, 0.715 * s},
		}, template)
	}
	return downsample(img, scale)
}

func drawHourglass(img *image.RGBA, x, y, w, h, lineWidth float64, stroke color.RGBA, fill color.RGBA) {
	topLeft := point{x, y}
	topRight := point{x + w, y}
	neck := point{x + w*0.5, y + h*0.50}
	bottomLeft := point{x, y + h}
	bottomRight := point{x + w, y + h}

	if fill.A != 0 {
		drawPolygon(img, []point{
			{x + w*0.31, y + h*0.25},
			{x + w*0.69, y + h*0.25},
			neck,
		}, fill)
		drawPolygon(img, []point{
			{x + w*0.28, y + h*0.80},
			{x + w*0.72, y + h*0.80},
			{x + w*0.50, y + h*0.56},
		}, fill)
	}

	drawLine(img, topLeft, topRight, lineWidth, stroke)
	drawLine(img, bottomLeft, bottomRight, lineWidth, stroke)
	drawLine(img, topLeft, neck, lineWidth, stroke)
	drawLine(img, topRight, neck, lineWidth, stroke)
	drawLine(img, neck, bottomLeft, lineWidth, stroke)
	drawLine(img, neck, bottomRight, lineWidth, stroke)
	drawCircle(img, neck.x, neck.y, lineWidth*0.62, stroke)
}

func drawRoundedRect(img *image.RGBA, x, y, w, h, r float64, c color.RGBA) {
	minX, maxX := int(math.Floor(x)), int(math.Ceil(x+w))
	minY, maxY := int(math.Floor(y)), int(math.Ceil(y+h))
	for py := minY; py < maxY; py++ {
		for px := minX; px < maxX; px++ {
			fx, fy := float64(px)+0.5, float64(py)+0.5
			cx := clamp(fx, x+r, x+w-r)
			cy := clamp(fy, y+r, y+h-r)
			if math.Hypot(fx-cx, fy-cy) <= r {
				img.SetRGBA(px, py, c)
			}
		}
	}
}

func drawLine(img *image.RGBA, a, b point, width float64, c color.RGBA) {
	r := width / 2
	minX := int(math.Floor(math.Min(a.x, b.x) - r - 1))
	maxX := int(math.Ceil(math.Max(a.x, b.x) + r + 1))
	minY := int(math.Floor(math.Min(a.y, b.y) - r - 1))
	maxY := int(math.Ceil(math.Max(a.y, b.y) + r + 1))
	dx, dy := b.x-a.x, b.y-a.y
	lenSq := dx*dx + dy*dy
	for py := minY; py <= maxY; py++ {
		for px := minX; px <= maxX; px++ {
			fx, fy := float64(px)+0.5, float64(py)+0.5
			t := ((fx-a.x)*dx + (fy-a.y)*dy) / lenSq
			t = clamp(t, 0, 1)
			cx, cy := a.x+t*dx, a.y+t*dy
			if math.Hypot(fx-cx, fy-cy) <= r {
				img.SetRGBA(px, py, c)
			}
		}
	}
}

func drawCircle(img *image.RGBA, cx, cy, r float64, c color.RGBA) {
	minX, maxX := int(math.Floor(cx-r)), int(math.Ceil(cx+r))
	minY, maxY := int(math.Floor(cy-r)), int(math.Ceil(cy+r))
	for py := minY; py <= maxY; py++ {
		for px := minX; px <= maxX; px++ {
			if math.Hypot(float64(px)+0.5-cx, float64(py)+0.5-cy) <= r {
				img.SetRGBA(px, py, c)
			}
		}
	}
}

func drawPolygon(img *image.RGBA, pts []point, c color.RGBA) {
	if len(pts) < 3 {
		return
	}
	minX, minY := pts[0].x, pts[0].y
	maxX, maxY := minX, minY
	for _, p := range pts[1:] {
		minX, maxX = math.Min(minX, p.x), math.Max(maxX, p.x)
		minY, maxY = math.Min(minY, p.y), math.Max(maxY, p.y)
	}
	for py := int(math.Floor(minY)); py <= int(math.Ceil(maxY)); py++ {
		for px := int(math.Floor(minX)); px <= int(math.Ceil(maxX)); px++ {
			if pointInPolygon(float64(px)+0.5, float64(py)+0.5, pts) {
				img.SetRGBA(px, py, c)
			}
		}
	}
}

func pointInPolygon(x, y float64, pts []point) bool {
	inside := false
	j := len(pts) - 1
	for i := range pts {
		if (pts[i].y > y) != (pts[j].y > y) &&
			x < (pts[j].x-pts[i].x)*(y-pts[i].y)/(pts[j].y-pts[i].y)+pts[i].x {
			inside = !inside
		}
		j = i
	}
	return inside
}

func downsample(src *image.RGBA, scale int) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx()/scale, b.Dy()/scale))
	for y := 0; y < dst.Bounds().Dy(); y++ {
		for x := 0; x < dst.Bounds().Dx(); x++ {
			var r, g, b, a uint32
			for sy := 0; sy < scale; sy++ {
				for sx := 0; sx < scale; sx++ {
					cr, cg, cb, ca := src.At(x*scale+sx, y*scale+sy).RGBA()
					r += cr
					g += cg
					b += cb
					a += ca
				}
			}
			div := uint32(scale * scale)
			dst.SetRGBA(x, y, color.RGBA{uint8((r / div) >> 8), uint8((g / div) >> 8), uint8((b / div) >> 8), uint8((a / div) >> 8)})
		}
	}
	return dst
}

func writePNG(path string, img image.Image) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		log.Fatal(err)
	}
	file, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	if err := png.Encode(file, img); err != nil {
		log.Fatal(err)
	}
}

func clamp(v, min, max float64) float64 {
	return math.Max(min, math.Min(max, v))
}
