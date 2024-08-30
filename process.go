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

type ProcessResult struct {
	Image      image.Image
	Embeddings []float32
}

func Process(ctx context.Context, emb embeddings.Embedder, r io.ReadSeeker) (*ProcessResult, error) {

	t1 := time.Now()

	defer func() {
		slog.Debug("Time to process record", "time", time.Since(t1))
	}()

	im, _, err := image.Decode(r)

	if err != nil {
		return nil, fmt.Errorf("Failed to decode image, %w", err)
	}

	_, err = r.Seek(0, 0)

	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(r)

	if err != nil {
		return nil, err
	}

	embeddings, err := emb.ImageEmbeddings32(ctx, body)

	if err != nil {
		return nil, fmt.Errorf("Failed to generate embeddings, %w", err)
	}

	pr := &ProcessResult{
		Image:      im,
		Embeddings: embeddings,
	}

	return pr, nil
}
