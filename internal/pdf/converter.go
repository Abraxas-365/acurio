package pdf

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"

	"github.com/gen2brain/go-fitz" // Lightweight PDF renderer
)

// ConvertPDFToImages converts PDF pages to JPEG images
func ConvertPDFToImages(pdfData []byte) ([][]byte, error) {
	doc, err := fitz.NewFromMemory(pdfData)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF: %w", err)
	}
	defer doc.Close()

	pageCount := doc.NumPage()
	images := make([][]byte, 0, pageCount)

	for i := 0; i < pageCount; i++ {
		img, err := doc.Image(i)
		if err != nil {
			return nil, fmt.Errorf("failed to render page %d: %w", i, err)
		}

		// Convert to JPEG bytes
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
			return nil, fmt.Errorf("failed to encode page %d: %w", i, err)
		}

		images = append(images, buf.Bytes())
	}

	return images, nil
}

// DetectImageFormat detects if data is already an image
func DetectImageFormat(data []byte) (string, error) {
	_, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	return format, nil
}

// ConvertImageToJPEG converts any image format to JPEG
func ConvertImageToJPEG(imageData []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	return buf.Bytes(), nil
}
