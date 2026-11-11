package export

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
)

// Растровые иллюстрации для Word-отчёта: схемы сечений таблицы 2 и рисунок 1.
// Рисуем сами, чтобы не тянуть шрифтовые и графические зависимости; текст
// подписей набираем встроенным растровым шрифтом 5×7 (цифры и нужные латинские
// буквы) — этого достаточно для подписей h, b, s, t, R и чисел на осях.

// font5x7 — глифы 5×7 бит: по строке на элемент, старший бит слева.
var font5x7 = map[rune][7]byte{
	'0': {0b01110, 0b10001, 0b10011, 0b10101, 0b11001, 0b10001, 0b01110},
	'1': {0b00100, 0b01100, 0b00100, 0b00100, 0b00100, 0b00100, 0b01110},
	'2': {0b01110, 0b10001, 0b00001, 0b00010, 0b00100, 0b01000, 0b11111},
	'3': {0b11110, 0b00001, 0b00001, 0b01110, 0b00001, 0b00001, 0b11110},
	'4': {0b00010, 0b00110, 0b01010, 0b10010, 0b11111, 0b00010, 0b00010},
	'5': {0b11111, 0b10000, 0b11110, 0b00001, 0b00001, 0b10001, 0b01110},
	'6': {0b00110, 0b01000, 0b10000, 0b11110, 0b10001, 0b10001, 0b01110},
	'7': {0b11111, 0b00001, 0b00010, 0b00100, 0b01000, 0b01000, 0b01000},
	'8': {0b01110, 0b10001, 0b10001, 0b01110, 0b10001, 0b10001, 0b01110},
	'9': {0b01110, 0b10001, 0b10001, 0b01111, 0b00001, 0b00010, 0b01100},
	'h': {0b10000, 0b10000, 0b10110, 0b11001, 0b10001, 0b10001, 0b10001},
	'b': {0b10000, 0b10000, 0b10110, 0b11001, 0b10001, 0b11001, 0b10110},
	's': {0b01110, 0b10001, 0b10000, 0b01110, 0b00001, 0b10001, 0b01110},
	't': {0b00100, 0b00100, 0b11111, 0b00100, 0b00100, 0b00101, 0b00010},
	'R': {0b11110, 0b10001, 0b10001, 0b11110, 0b10100, 0b10010, 0b10001},
	'F': {0b11111, 0b10000, 0b10000, 0b11110, 0b10000, 0b10000, 0b10000},
	'P': {0b11110, 0b10001, 0b10001, 0b11110, 0b10000, 0b10000, 0b10000},
	'V': {0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b01010, 0b00100},
	'M': {0b10001, 0b11011, 0b10101, 0b10101, 0b10001, 0b10001, 0b10001},
	'L': {0b10000, 0b10000, 0b10000, 0b10000, 0b10000, 0b10000, 0b11111},
	'S': {0b01111, 0b10000, 0b10000, 0b01110, 0b00001, 0b00001, 0b11110},
	'q': {0b01111, 0b10001, 0b10001, 0b01111, 0b00001, 0b00001, 0b00001},
	'Z': {0b11111, 0b00001, 0b00010, 0b00100, 0b01000, 0b10000, 0b11111},
	'x': {0b10001, 0b01010, 0b00100, 0b01010, 0b10001, 0b00000, 0b00000},
	'.': {0b00000, 0b00000, 0b00000, 0b00000, 0b00000, 0b01100, 0b01100},
	'-': {0b00000, 0b00000, 0b00000, 0b11111, 0b00000, 0b00000, 0b00000},
	' ': {0b00000, 0b00000, 0b00000, 0b00000, 0b00000, 0b00000, 0b00000},
}

// drawText печатает строку встроенным шрифтом; scale — размер пикселя глифа.
func drawText(img *image.RGBA, x, y, scale int, text string, col color.Color) {
	cx := x
	for _, r := range text {
		glyph, ok := font5x7[r]
		if !ok {
			cx += 6 * scale
			continue
		}
		for row := 0; row < 7; row++ {
			bits := glyph[row]
			for colIdx := 0; colIdx < 5; colIdx++ {
				if bits&(1<<uint(4-colIdx)) == 0 {
					continue
				}
				for dy := 0; dy < scale; dy++ {
					for dx := 0; dx < scale; dx++ {
						px, py := cx+colIdx*scale+dx, y+row*scale+dy
						if px >= 0 && py >= 0 && px < img.Bounds().Dx() && py < img.Bounds().Dy() {
							img.Set(px, py, col)
						}
					}
				}
			}
		}
		cx += 6 * scale
	}
}

func textWidth(scale int, text string) int {
	n := 0
	for _, r := range text {
		if _, ok := font5x7[r]; ok {
			n++
		}
	}
	return n * 6 * scale
}

// line рисует отрезок (алгоритм Брезенхэма).
func rasterLine(img *image.RGBA, x1, y1, x2, y2 int, col color.Color) {
	dx, dy := x2-x1, y2-y1
	steps := int(math.Abs(float64(dx)))
	if int(math.Abs(float64(dy))) > steps {
		steps = int(math.Abs(float64(dy)))
	}
	for s := 0; s <= steps; s++ {
		t := 0.0
		if steps > 0 {
			t = float64(s) / float64(steps)
		}
		x, y := x1+int(float64(dx)*t), y1+int(float64(dy)*t)
		if x >= 0 && y >= 0 && x < img.Bounds().Dx() && y < img.Bounds().Dy() {
			img.Set(x, y, col)
		}
	}
}

func newCanvas(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
	return img
}

func encodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// sketchPNG — схема сечения для таблицы 2: повторяет геометрию PDF-схемы.
func sketchPNG(kind string) ([]byte, error) {
	const w, h = 300, 150
	img := newCanvas(w, h)
	ink := color.RGBA{60, 60, 60, 255}
	dim := color.RGBA{150, 150, 150, 255}
	fx := func(v float64) int { return int(v) }
	switch kind {
	case "ibeam_slope", "ibeam":
		slope := kind == "ibeam_slope"
		hh, bb := 120.0, 90.0
		tf, tw := hh*0.14, bb*0.15
		cx := 150.0
		y0 := 15.0
		y1 := y0 + hh
		sl := 0.0
		if slope {
			sl = tf * 0.8
		}
		pts := [][2]float64{
			{cx - bb/2, y0}, {cx + bb/2, y0}, {cx + bb/2, y0 + tf},
			{cx + tw/2, y0 + tf + sl}, {cx + tw/2, y1 - tf - sl}, {cx + bb/2, y1 - tf},
			{cx + bb/2, y1}, {cx - bb/2, y1}, {cx - bb/2, y1 - tf},
			{cx - tw/2, y1 - tf - sl}, {cx - tw/2, y0 + tf + sl}, {cx - bb/2, y0 + tf},
		}
		for i := range pts {
			p1, p2 := pts[i], pts[(i+1)%len(pts)]
			rasterLine(img, fx(p1[0]), fx(p1[1]), fx(p2[0]), fx(p2[1]), ink)
		}
		// размерные линии h и b с подписями
		rasterLine(img, fx(cx-bb/2-2.2), fx(y0), fx(cx-bb/2-2.2), fx(y1), dim)
		drawText(img, fx(cx-bb/2-6.4), fx(y0+hh/2-1.2), 2, "h", ink)
		rasterLine(img, fx(cx-bb/2), fx(y1+2.2), fx(cx+bb/2), fx(y1+2.2), dim)
		drawText(img, fx(cx-1.6), fx(y1+2.6), 2, "b", ink)
		drawText(img, fx(cx+tw/2+1.6), fx(y0+hh*0.5-3.0), 2, "s", ink)
		drawText(img, fx(cx+bb/2+1.0), fx(y0+tf/2-2.6), 2, "t", ink)
		if !slope {
			drawText(img, fx(cx+tw/2+1.4), fx(y1-tf-4.2), 2, "R", ink)
		}
	default:
		side := 110.0
		rounded := kind == "tube_sq"
		cx, cy := 150.0, 75.0
		x0, y0 := cx-side/2, cy-side/2
		wall := side * 0.13
		ch := 0.0
		if rounded {
			ch = side * 0.20
		}
		tube := func(px, py, ps, pc float64) {
			pts := [][2]float64{
				{px + pc, py}, {px + ps - pc, py}, {px + ps, py + pc},
				{px + ps, py + ps - pc}, {px + ps - pc, py + ps}, {px + pc, py + ps},
				{px, py + ps - pc}, {px, py + pc},
			}
			for i := range pts {
				p1, p2 := pts[i], pts[(i+1)%len(pts)]
				rasterLine(img, fx(p1[0]), fx(p1[1]), fx(p2[0]), fx(p2[1]), ink)
			}
		}
		tube(x0, y0, side, ch)
		tube(x0+wall, y0+wall, side-2*wall, ch*0.6)
		rasterLine(img, fx(x0), fx(y0+side+2.2), fx(x0+side), fx(y0+side+2.2), dim)
		drawText(img, fx(cx-1.6), fx(y0+side+2.6), 2, "h", ink)
		drawText(img, fx(x0+side+1.6), fx(y0+side*0.5-1.2), 2, "s", ink)
		if rounded {
			drawText(img, fx(x0+4), fx(y0+side*0.55), 2, "R", ink)
		}
	}
	return encodePNG(img)
}

// chartPNG — рисунок 1: предел огнестойкости от приведённой толщины металла.
func (d *docxReport) chartPNG() ([]byte, error) {
	const w, h = 900, 620
	const left, right, top, bottom = 90, 60, 30, 80
	img := newCanvas(w, h)
	plotW, plotH := w-left-right, h-top-bottom
	x0, y0 := left, top+plotH
	grid := color.RGBA{215, 215, 215, 255}
	axis := color.RGBA{110, 110, 110, 255}
	inkCol := color.RGBA{60, 60, 60, 255}

	// Ось X — по данным: максимальный δпр, при котором кривая упирается в
	// потолок 250 мин, плюс 15 % запаса (как в PDF-отчёте).
	maxX := 10.0
	for _, th := range []float64{30, 40, 50, 60, 70, 80, 90} {
		for step := 0.0; step <= 60; step += 0.5 {
			if d.data.factRFor(th, step) >= 250 {
				if step > maxX {
					maxX = step
				}
				break
			}
		}
	}
	maxX = math.Ceil(maxX*1.15/5) * 5
	if maxX < 15 {
		maxX = 15
	}
	if maxX > 50 {
		maxX = 50
	}
	const maxY = 250.0
	for i := 0; i <= 5; i++ {
		x := x0 + plotW*i/5
		rasterLine(img, x, top, x, y0, grid)
		label := fmtNum(maxX*float64(i)/5, 0)
		drawText(img, x-textWidth(1, label)/2, y0+12, 1, label, inkCol)
	}
	for i := 0; i <= 5; i++ {
		y := y0 - plotH*i/5
		rasterLine(img, x0, y, x0+plotW, y, grid)
		label := []string{"0", "50", "100", "150", "200", "250"}[i]
		drawText(img, x0-14-textWidth(1, label), y-3, 1, label, inkCol)
	}
	rasterLine(img, x0, top, x0+plotW, top, axis)
	rasterLine(img, x0, y0, x0+plotW, y0, axis)
	rasterLine(img, x0, top, x0, y0, axis)
	rasterLine(img, x0+plotW, top, x0+plotW, y0, axis)

	thicknesses := []float64{30, 40, 50, 60, 70, 80, 90}
	palette := []color.RGBA{
		{63, 81, 181, 255}, {0, 137, 123, 255}, {67, 160, 71, 255}, {251, 192, 45, 255},
		{245, 124, 0, 255}, {229, 57, 53, 255}, {142, 36, 170, 255},
	}
	for ti, th := range thicknesses {
		col := palette[ti%len(palette)]
		px, py, started := 0, 0, false
		for step := 0.0; step <= maxX; step += 0.25 {
			r := d.data.factRFor(th, step)
			if r <= 0 {
				started = false
				continue
			}
			x := x0 + int(float64(plotW)*step/maxX)
			y := y0 - int(float64(plotH)*math.Min(r, maxY)/maxY)
			if started {
				rasterLine(img, px, py, x, y, col)
				rasterLine(img, px, py+1, x, y+1, col)
			}
			px, py, started = x, y, true
		}
		// подпись толщины справа от графика — цветом её линии
		drawText(img, x0+plotW+8, y0-plotH+ti*28, 1, fmtNum(th, 0), palette[ti%len(palette)])
	}
	// ось X подписана растром, ось Y — в подписи рисунка
	drawText(img, x0+plotW/2-textWidth(1, "x 10-50")/2, y0+26, 1, "", inkCol)
	// стрелки осей
	rasterLine(img, x0+plotW, top, x0+plotW-8, top-4, axis)
	rasterLine(img, x0+plotW, top, x0+plotW-8, top+4, axis)
	rasterLine(img, x0, top, x0-4, top+8, axis)
	rasterLine(img, x0, top, x0+4, top+8, axis)
	return encodePNG(img)
}
