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
	resp, err := g.client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Input: openai.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: []string{text},
		},
		Model: "text-embedding-3-small", // Cost-effective and performant
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

	resp, err := g.client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Input: openai.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: texts,
		},
		Model: "text-embedding-3-small",
	})

	if err != nil {
		return nil, fmt.Errorf("failed to generate embeddings: %w", err)
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
