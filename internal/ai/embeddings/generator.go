package embeddings

import (
	"context"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// EmbeddingsGenerator handles creating embeddings for semantic search
type EmbeddingsGenerator struct {
	client *openai.Client
}

// NewEmbeddingsGenerator creates a new embeddings generator
func NewEmbeddingsGenerator(apiKey string) *EmbeddingsGenerator {
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
	)

	return &EmbeddingsGenerator{
		client: &client,
	}
}

// GenerateEmbedding creates an embedding vector for text
func (g *EmbeddingsGenerator) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	// Send as array with single element (works consistently)
	resp, err := g.client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Input: openai.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: []string{text},
		},
		Model: openai.EmbeddingModelTextEmbedding3Small,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data returned")
	}

	// Convert []float64 to []float32
	embedding64 := resp.Data[0].Embedding
	embedding32 := make([]float32, len(embedding64))
	for i, v := range embedding64 {
		embedding32[i] = float32(v)
	}

	return embedding32, nil
}

// GenerateBatchEmbeddings creates embeddings for multiple texts
func (g *EmbeddingsGenerator) GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("no texts provided")
	}

	// Filter out empty strings
	validTexts := make([]string, 0, len(texts))
	for _, text := range texts {
		if text != "" {
			validTexts = append(validTexts, text)
		}
	}

	if len(validTexts) == 0 {
		return nil, fmt.Errorf("all texts are empty")
	}

	resp, err := g.client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Input: openai.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: validTexts,
		},
		Model: openai.EmbeddingModelTextEmbedding3Small,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to generate embeddings: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data returned")
	}

	embeddings := make([][]float32, len(resp.Data))
	for i, data := range resp.Data {
		embedding32 := make([]float32, len(data.Embedding))
		for j, v := range data.Embedding {
			embedding32[j] = float32(v)
		}
		embeddings[i] = embedding32
	}

	return embeddings, nil
}
