package applicationsrv

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Abraxas-365/relay/internal/ai/embeddings"
	"github.com/Abraxas-365/relay/internal/ai/resumeparser"
	"github.com/Abraxas-365/relay/internal/pdf"
	"github.com/Abraxas-365/relay/pkg/kernel"
)

// ResumeProcessor handles resume parsing and embedding generation
type ResumeProcessor struct {
	parser   *resumeparser.ResumeParser
	embedder *embeddings.EmbeddingsGenerator
}

// NewResumeProcessor creates a new resume processor
func NewResumeProcessor(openaiAPIKey string) *ResumeProcessor {
	return &ResumeProcessor{
		parser:   resumeparser.NewResumeParser(openaiAPIKey),
		embedder: embeddings.NewEmbeddingsGenerator(openaiAPIKey),
	}
}

// ProcessResume processes a resume file (PDF or image) and returns summary + embedding
func (rp *ResumeProcessor) ProcessResume(
	ctx context.Context,
	fileData []byte,
	contentType string,
) (kernel.ResumeSummary, kernel.ResumeEmbedding, error) {

	var resumeData *resumeparser.ResumeData
	var err error

	// Determine file type and process accordingly
	if isPDFType(contentType) {
		// Convert PDF to images
		images, err := pdf.ConvertPDFToImages(fileData)
		if err != nil {
			return "", nil, fmt.Errorf("failed to convert PDF to images: %w", err)
		}

		// Parse using Vision API
		resumeData, err = rp.parser.ParseResumeFromMultiplePages(ctx, images)
		if err != nil {
			return "", nil, fmt.Errorf("failed to parse PDF resume: %w", err)
		}
	} else if isImageType(contentType) {
		// Convert image to JPEG if needed
		imageData := fileData
		if !isJPEGType(contentType) {
			imageData, err = pdf.ConvertImageToJPEG(fileData)
			if err != nil {
				return "", nil, fmt.Errorf("failed to convert image: %w", err)
			}
		}

		// Parse using Vision API
		resumeData, err = rp.parser.ParseResumeFromImage(ctx, imageData)
		if err != nil {
			return "", nil, fmt.Errorf("failed to parse image resume: %w", err)
		}
	} else {
		return "", nil, fmt.Errorf("unsupported file type: %s", contentType)
	}

	// Convert to JSON string for storage
	summaryJSON, err := json.Marshal(resumeData)
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal resume data: %w", err)
	}

	// Create text for embedding
	embeddingText := resumeData.FormatResumeForEmbedding()

	// Generate embedding
	embedding, err := rp.embedder.GenerateEmbedding(ctx, embeddingText)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	return kernel.ResumeSummary(summaryJSON), kernel.ResumeEmbedding(embedding), nil
}

// Helper functions
func isPDFType(contentType string) bool {
	return strings.Contains(strings.ToLower(contentType), "pdf")
}

func isImageType(contentType string) bool {
	imageTypes := []string{"image/jpeg", "image/jpg", "image/png", "image/webp", "image/gif"}
	lower := strings.ToLower(contentType)
	for _, t := range imageTypes {
		if strings.Contains(lower, t) {
			return true
		}
	}
	return false
}

func isJPEGType(contentType string) bool {
	lower := strings.ToLower(contentType)
	return strings.Contains(lower, "jpeg") || strings.Contains(lower, "jpg")
}
