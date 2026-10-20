package colorpkg

import (
	"image/color"
	"math"
)

// ... (existing functions RGBToHSL, HSLToRGB, hueToRGB, Recolor) ...

// RecolorMultiply applies the targetColor to the sourceColor using the Multiply blend mode.
// White pixels will become the target color, black will stay black, and grey will be darkened target color.
// Alpha is preserved.
func RecolorMultiply(sourceColor color.Color, targetColor color.Color) color.Color {
	sr, sg, sb, sa := sourceColor.RGBA()
	tr, tg, tb, _ := targetColor.RGBA()

	if sa == 0 {
		return color.NRGBA{0, 0, 0, 0}
	}

	// Un-premultiply source and convert to 0..1
	saf := float64(sa) / 65535.0
	srf := (float64(sr) / 65535.0) / saf
	sgf := (float64(sg) / 65535.0) / saf
	sbf := (float64(sb) / 65535.0) / saf

	// Un-premultiply target (target color is usually fully opaque)
	_, _, _, ta := targetColor.RGBA()
	trf := float64(tr) / 65535.0
	tgf := float64(tg) / 65535.0
	tbf := float64(tb) / 65535.0
	if ta > 0 {
		taf := float64(ta) / 65535.0
		trf /= taf
		tgf /= taf
		tbf /= taf
	}

	// Multiply blend mode: Result = Source * Target
	nr := srf * trf
	ng := sgf * tgf
	nb := sbf * tbf

	// Convert back to 0..255
	return color.NRGBA{
		R: uint8(math.Round(nr * 255.0)),
		G: uint8(math.Round(ng * 255.0)),
		B: uint8(math.Round(nb * 255.0)),
		A: uint8(math.Round(saf * 255.0)),
	}
}
