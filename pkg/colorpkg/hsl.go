package colorpkg

import (
	"image/color"
	"math"
)

// RGBToHSL converts an RGB color to HSL.
// r, g, b are in the range 0..1
// h, s, l are in the range 0..1
func RGBToHSL(r, g, b float64) (h, s, l float64) {
	max := math.Max(r, math.Max(g, b))
	min := math.Min(r, math.Min(g, b))
	l = (max + min) / 2

	if max == min {
		h = 0
		s = 0
	} else {
		d := max - min
		if l > 0.5 {
			s = d / (2 - max - min)
		} else {
			s = d / (max + min)
		}
		switch max {
		case r:
			h = (g - b) / d
			if g < b {
				h += 6
			}
		case g:
			h = (b-r)/d + 2
		case b:
			h = (r-g)/d + 4
		}
		h /= 6
	}
	return
}

// HSLToRGB converts an HSL color to RGB.
// h, s, l are in the range 0..1
// r, g, b are in the range 0..1
func HSLToRGB(h, s, l float64) (r, g, b float64) {
	if s == 0 {
		r = l
		g = l
		b = l
	} else {
		var q float64
		if l < 0.5 {
			q = l * (1 + s)
		} else {
			q = l + s - l*s
		}
		p := 2*l - q
		r = hueToRGB(p, q, h+1.0/3.0)
		g = hueToRGB(p, q, h)
		b = hueToRGB(p, q, h-1.0/3.0)
	}
	return
}

func hueToRGB(p, q, t float64) float64 {
	if t < 0 {
		t += 1
	}
	if t > 1 {
		t -= 1
	}
	if t < 1.0/6.0 {
		return p + (q-p)*6*t
	}
	if t < 1.0/2.0 {
		return q
	}
	if t < 2.0/3.0 {
		return p + (q-p)*(2.0/3.0-t)*6
	}
	return p
}

// Recolor applies the hue and saturation of targetColor to the lightness of sourceColor.
// Returns a new color.Color with the modified RGB and original Alpha.
func Recolor(sourceColor color.Color, targetColor color.Color) color.Color {
	sr, sg, sb, sa := sourceColor.RGBA()
	tr, tg, tb, _ := targetColor.RGBA()

	// If the alpha is 0, pixel is transparent. Return transparent black (or just transparent).
	if sa == 0 {
		return color.NRGBA{0, 0, 0, 0}
	}

	// Convert 0..65535 to 0..1
	srf := float64(sr) / 65535.0
	sgf := float64(sg) / 65535.0
	sbf := float64(sb) / 65535.0
	saf := float64(sa) / 65535.0

	// Un-premultiply source
	srf /= saf
	sgf /= saf
	sbf /= saf

	trf := float64(tr) / 65535.0
	tgf := float64(tg) / 65535.0
	tbf := float64(tb) / 65535.0

	// Un-premultiply target (target color is usually fully opaque)
	_, _, _, ta := targetColor.RGBA()
	if ta > 0 {
		taf := float64(ta) / 65535.0
		trf /= taf
		tgf /= taf
		tbf /= taf
	}

	// Get Target Hue and Saturation
	th, ts, _ := RGBToHSL(trf, tgf, tbf)

	// Get Source Lightness
	_, _, sl := RGBToHSL(srf, sgf, sbf)

	// Combine
	nr, ng, nb := HSLToRGB(th, ts, sl)

	// Convert back to 0..255
	return color.NRGBA{
		R: uint8(math.Round(nr * 255.0)),
		G: uint8(math.Round(ng * 255.0)),
		B: uint8(math.Round(nb * 255.0)),
		A: uint8(math.Round(saf * 255.0)),
	}
}
