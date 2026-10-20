package imagepkg

import (
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"recol/pkg/colorpkg"
)

// ProcessImage loads an image, recolors it using targetColor, and saves it.
func ProcessImage(filePath string, targetColor color.Color, mode string) error {
	// Open the original file
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}

	// Decode the image
	img, format, err := image.Decode(file)
	file.Close()
	if err != nil {
		return err
	}

	// Create a new image of the same size
	bounds := img.Bounds()
	newImg := image.NewNRGBA(bounds)

	// Recolor pixel by pixel
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			originalColor := img.At(x, y)
			var newColor color.Color
			if mode == "multiply" {
				newColor = colorpkg.RecolorMultiply(originalColor, targetColor)
			} else {
				newColor = colorpkg.Recolor(originalColor, targetColor)
			}
			newImg.Set(x, y, newColor)
		}
	}

	// Save to a temporary file first
	tempFilePath := filePath + ".tmp"
	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		return err
	}

	if format == "jpeg" {
		err = jpeg.Encode(tempFile, newImg, &jpeg.Options{Quality: 90})
	} else {
		err = png.Encode(tempFile, newImg)
	}

	tempFile.Close()
	if err != nil {
		os.Remove(tempFilePath)
		return err
	}

	// Rename temp to original
	err = os.Rename(tempFilePath, filePath)
	if err != nil {
		return err
	}

	return nil
}

// IsSupported returns true if the file extension is supported.
func IsSupported(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".png" || ext == ".jpg" || ext == ".jpeg"
}
