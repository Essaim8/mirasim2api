// genicon 生成 mirasim2api 的托盘模板图标与 macOS iconset（iconutil 源图）。
// 图形为「信号柱」：三根渐高圆角柱，托盘版纯黑透明底（template），
// App 图标版紫蓝渐变 squircle 底 + 白色柱。
//
//	go run ./deploy/macos/genicon -trayout ../../cmd/desktop/assets -iconset /tmp/AppIcon.iconset
package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"

	"image/png"
)

func main() {
	trayOut := flag.String("trayout", "", "目录：写入 trayTemplate.png(22) 与 trayTemplate@2x.png(44)")
	iconsetOut := flag.String("iconset", "", "目录：写入 icon_*.png（iconutil 输入）")
	icoOut := flag.String("icoout", "", "文件：写入 Windows .ico（16/32/48/256 PNG 压缩帧）")
	flag.Parse()
	if *trayOut == "" && *iconsetOut == "" && *icoOut == "" {
		fmt.Fprintln(os.Stderr, "用法: genicon [-trayout DIR] [-iconset DIR] [-icoout FILE]")
		os.Exit(2)
	}
	if *trayOut != "" {
		for _, spec := range []struct {
			name string
			px   int
		}{{"trayTemplate.png", 22}, {"trayTemplate@2x.png", 44}} {
			img := renderGlyph(spec.px, color.NRGBA{0, 0, 0, 255}, nil)
			if err := writePNG(filepath.Join(*trayOut, spec.name), img); err != nil {
				fatal(err)
			}
		}
	}
	if *iconsetOut != "" {
		for _, spec := range []struct {
			name string
			px   int
		}{
			{"icon_16x16.png", 16}, {"icon_16x16@2x.png", 32},
			{"icon_32x32.png", 32}, {"icon_32x32@2x.png", 64},
			{"icon_128x128.png", 128}, {"icon_128x128@2x.png", 256},
			{"icon_256x256.png", 256}, {"icon_256x256@2x.png", 512},
			{"icon_512x512.png", 512}, {"icon_512x512@2x.png", 1024},
		} {
			img := renderGlyph(spec.px, color.NRGBA{255, 255, 255, 255}, gradientSquircle)
			if err := writePNG(filepath.Join(*iconsetOut, spec.name), img); err != nil {
				fatal(err)
			}
		}
	}
	if *icoOut != "" {
		if err := writeICO(*icoOut, []int{16, 32, 48, 256}); err != nil {
			fatal(err)
		}
	}
}

// writeICO 产出 Windows .ico：多尺寸 PNG 压缩帧（Vista+ 与 Win32 均支持）。
func writeICO(path string, sizes []int) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	frames := make([][]byte, 0, len(sizes))
	for _, px := range sizes {
		var buf bytes.Buffer
		img := renderGlyph(px, color.NRGBA{255, 255, 255, 255}, gradientSquircle)
		if err := png.Encode(&buf, img); err != nil {
			return err
		}
		frames = append(frames, buf.Bytes())
	}

	// ICONDIR：reserved(2) type=1(2) count(2)
	hdr := make([]byte, 6)
	binary.LittleEndian.PutUint16(hdr[2:], 1)
	binary.LittleEndian.PutUint16(hdr[4:], uint16(len(frames)))
	if _, err := f.Write(hdr); err != nil {
		return err
	}
	offset := 6 + len(frames)*16
	for i, px := range sizes {
		entry := make([]byte, 16)
		// 宽高字段 0 代表 256
		w, h := byte(px), byte(px)
		if px >= 256 {
			w, h = 0, 0
		}
		entry[0], entry[1] = w, h
		binary.LittleEndian.PutUint16(entry[4:], 1)  // planes
		binary.LittleEndian.PutUint16(entry[6:], 32) // bitCount
		binary.LittleEndian.PutUint32(entry[8:], uint32(len(frames[i])))
		binary.LittleEndian.PutUint32(entry[12:], uint32(offset))
		if _, err := f.Write(entry); err != nil {
			return err
		}
		offset += len(frames[i])
	}
	for _, frame := range frames {
		if _, err := f.Write(frame); err != nil {
			return err
		}
	}
	return nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "genicon:", err)
	os.Exit(1)
}

func writePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// renderGlyph 画三根渐高圆角柱。bg 非 nil 时先铺满背景（用于 App 图标）。
func renderGlyph(px int, fg color.NRGBA, bg func(x, y, px float64) color.NRGBA) *image.NRGBA {
	const ss = 4 // 超采样抗锯齿
	s := px * ss
	img := image.NewNRGBA(image.Rect(0, 0, s, s))
	if bg != nil {
		for y := 0; y < s; y++ {
			for x := 0; x < s; x++ {
				img.SetNRGBA(x, y, bg(float64(x)/ss, float64(y)/ss, float64(px)))
			}
		}
	}
	fs := float64(s)
	// 柱参数：宽 18%，间距 8%，整体居中，垂直占 64%（上下各留 18%）。
	barW := fs * 0.18
	gap := fs * 0.08
	total := barW*3 + gap*2
	x0 := (fs - total) / 2
	yTop := fs * 0.18
	yFull := fs * 0.64
	heights := []float64{0.45, 0.72, 1.0}
	for i, h := range heights {
		bh := yFull * h
		pill(img, x0+float64(i)*(barW+gap), yTop+yFull-bh, barW, bh, fg)
	}
	if ss == 1 {
		return img
	}
	return downsample(img, ss)
}

// pill 画圆头竖柱（宽度 w，半圆帽）。
func pill(img *image.NRGBA, x, y, w, h float64, c color.NRGBA) {
	r := w / 2
	b := img.Bounds()
	for py := b.Min.Y; py < b.Max.Y; py++ {
		for px := b.Min.X; px < b.Max.X; px++ {
			fx, fy := float64(px)+0.5, float64(py)+0.5
			if fy < y || fy > y+h || fx < x || fx > x+w {
				continue
			}
			in := true
			// 顶/底圆帽
			for _, cy := range [][2]float64{{y + r, -1}, {y + h - r, 1}} {
				if (cy[1] < 0 && fy < cy[0]) || (cy[1] > 0 && fy > cy[0]) {
					dx := fx - (x + r)
					dy := fy - cy[0]
					if dx*dx+dy*dy > r*r {
						in = false
					}
				}
			}
			if in {
				img.SetNRGBA(px, py, c)
			}
		}
	}
}

// gradientSquircle 返回 macOS 风格 squircle（超椭圆）+ 竖向渐变背景。
func gradientSquircle(x, y, px float64) color.NRGBA {
	nx := x/px*2 - 1
	ny := y/px*2 - 1
	// margin：整圆角图形内缩 6%
	const m = 0.88
	const n = 4.6 // 超椭圆指数，越大越接近圆角矩形
	if math.Pow(math.Abs(nx/m), n)+math.Pow(math.Abs(ny/m), n) > 1 {
		return color.NRGBA{}
	}
	t := y / px
	top := [3]float64{88, 101, 242}   // #5865F2
	bottom := [3]float64{49, 46, 129} // #312E81
	return color.NRGBA{
		uint8(top[0]*(1-t) + bottom[0]*t),
		uint8(top[1]*(1-t) + bottom[1]*t),
		uint8(top[2]*(1-t) + bottom[2]*t),
		255,
	}
}

// downsample 以 box 滤波将图缩小 ss 倍。
func downsample(src *image.NRGBA, ss int) *image.NRGBA {
	w := src.Bounds().Dx() / ss
	h := src.Bounds().Dy() / ss
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var r, g, b, a int
			for dy := 0; dy < ss; dy++ {
				for dx := 0; dx < ss; dx++ {
					p := src.NRGBAAt(x*ss+dx, y*ss+dy)
					r += int(p.R)
					g += int(p.G)
					b += int(p.B)
					a += int(p.A)
				}
			}
			n := ss * ss
			dst.SetNRGBA(x, y, color.NRGBA{uint8(r / n), uint8(g / n), uint8(b / n), uint8(a / n)})
		}
	}
	return dst
}
