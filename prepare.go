package similar

import (
	"context"
	"fmt"
	"image"
	"io"
	"log/slog"
	"time"

	"github.com/whosonfirst/go-dedupe/embeddings"
)

// PrepareResult is a struct containing the results derived by the `Prepare` method.
type PrepareResult struct {
	// Image is the `image.Image` instance that was decoded.
	Image      image.Image
	// Embeddings are the (vector) embeddings derived from 'Image'.
	Embeddings []float32
}

// Prepare() derives (vector) embeddings for an image file encoded in 'r' using 'emb'.
func Prepare(ctx context.Context, emb embeddings.Embedder, r io.ReadSeeker) (*PrepareResult, error) {

	t1 := time.Now()

	defer func() {
		slog.Debug("Time to prepare record", "time", time.Since(t1))
	}()

	im, _, err := image.Decode(r)

	if err != nil {
		return nil, fmt.Errorf("Failed to decode image, %w", err)
	}

	_, err = r.Seek(0, 0)

	if err != nil {
		return nil, fmt.Errorf("Failed to rewind reader, %w", err)
	}

	body, err := io.ReadAll(r)

	if err != nil {
		return nil, fmt.Errorf("Failed to read body, %w", err)
	}

	embeddings, err := emb.ImageEmbeddings32(ctx, body)

	if err != nil {
		return nil, fmt.Errorf("Failed to generate embeddings, %w", err)
	}

	pr := &PrepareResult{
		Image:      im,
		Embeddings: embeddings,
	}

	return pr, nil
}
