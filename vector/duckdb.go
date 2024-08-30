//go:build duckdb

package vector

// https://duckdb.org/2024/05/03/vector-similarity-search-vss.html
// https://duckdb.org/docs/api/go.html
// https://pkg.go.dev/github.com/marcboeker/go-duckdb
// https://github.com/marcboeker/go-duckdb/tree/main?tab=readme-ov-file#vendoring

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"time"

	_ "github.com/marcboeker/go-duckdb"
)

type DuckDBDatabase struct {
	// The underlying SQLite database used to store and query embeddings.
	vec_db     *sql.DB
	dimensions int
	// The maximum number of results for queries
	max_results int
	// The compression type to use for embeddings. Valid options are: quantize, matroyshka, none (default)
	compression string
	// If true that existing records are re-indexed. If not, they are skipped and left as-is.
	refresh      bool
	max_distance float32
}

func init() {

	ctx := context.Background()
	err := RegisterDatabase(ctx, "duckdb", NewDuckDBDatabase)

	if err != nil {
		panic(err)
	}
}

func NewDuckDBDatabase(ctx context.Context, uri string) (Database, error) {

	u, err := url.Parse(uri)

	if err != nil {
		return nil, fmt.Errorf("Failed to parse URI, %w", err)
	}

	q := u.Query()

	dimensions := 512 // mlx_clip
	max_distance := float32(5.0)
	max_results := 10
	refresh := false

	if q.Has("dimensions") {

		v, err := strconv.Atoi(q.Get("dimensions"))

		if err != nil {
			return nil, fmt.Errorf("Invalid ?dimensions= parameter, %w", err)
		}

		dimensions = v
	}

	if q.Has("max-distance") {

		v, err := strconv.ParseFloat(q.Get("max-distance"), 64)

		if err != nil {
			return nil, fmt.Errorf("Invalid ?max-distance= parameter, %w", err)
		}

		max_distance = float32(v)
	}

	if q.Has("max-results") {

		v, err := strconv.Atoi(q.Get("max-results"))

		if err != nil {
			return nil, fmt.Errorf("Invalid ?max-results= parameter, %w", err)
		}

		max_results = v
	}

	if q.Has("refresh") {

		v, err := strconv.ParseBool(q.Get("refresh"))

		if err != nil {
			return nil, fmt.Errorf("Invalid ?refresh= parameter, %w", err)
		}

		refresh = v
	}

	vec_db, err := sql.Open("duckdb", "")

	if err != nil {
		return nil, fmt.Errorf("Failed to open database connection, %w", err)
	}

	err = setupDuckDBDatabase(ctx, vec_db, dimensions)

	if err != nil {
		return nil, fmt.Errorf("Failed to setup database, %w", err)
	}

	if q.Has("max-conns") {

		v, err := strconv.Atoi(q.Get("max-conns"))

		if err != nil {
			return nil, err
		}

		vec_db.SetMaxOpenConns(v)
	}

	db := &DuckDBDatabase{
		vec_db:       vec_db,
		dimensions:   dimensions,
		max_distance: max_distance,
		max_results:  max_results,
		refresh:      refresh,
	}

	return db, nil
}

func (db *DuckDBDatabase) Add(ctx context.Context, rec *Image) error {

	id := rec.ID
	v, err := json.Marshal(rec.Embeddings)

	if err != nil {
		return err
	}

	q := "INSERT OR REPLACE INTO embeddings (id, vec) VALUES (?, ?)"
	slog.Debug(q)

	_, err = db.vec_db.ExecContext(ctx, q, id, string(v), string(v))

	if err != nil {
		return fmt.Errorf("Failed to add embeddings for %s, %w", id, err)
	}

	return nil
}

func (db *DuckDBDatabase) Query(ctx context.Context, r *Image) ([]*Match, error) {

	results := make([]*Match, 0)

	v, err := json.Marshal(r.Embeddings)

	if err != nil {
		return nil, err
	}

	q := fmt.Sprintf(`SELECT id, array_distance(vec, ?::FLOAT[%d]) AS distance
			  FROM embeddings WHERE distance <= %f AND id != ?
			  ORDER BY distance ASC LIMIT %d`,
		db.dimensions, db.max_distance, db.max_results)

	slog.Debug(q)

	t1 := time.Now()

	rows, err := db.vec_db.QueryContext(ctx, q, string(v), r.ID)

	if err != nil {
		return nil, fmt.Errorf("Failed to execute query, %w", err)
	}

	slog.Debug("Query context", "time", time.Since(t1))

	for rows.Next() {

		var id string
		var distance float64

		err = rows.Scan(&id, &distance)

		if err != nil {
			return nil, fmt.Errorf("Failed to scan row, %w", err)
		}

		r := &Match{
			ID:         id,
			Similarity: float32(distance),
		}

		slog.Debug("Result", "location id", id, "distance", distance)

		results = append(results, r)
	}

	slog.Debug("Query rows", "time", time.Since(t1))

	return results, nil
}

func (db *DuckDBDatabase) MeetsThreshold(ctx context.Context, qr *Match, threshold float64) (bool, error) {

	if float64(qr.Similarity) > threshold {
		return false, nil
	}

	return true, nil
}

func (db *DuckDBDatabase) Flush(ctx context.Context) error {
	return nil
}

func (db *DuckDBDatabase) Close(ctx context.Context) error {
	return db.vec_db.Close()
}

func setupDuckDBDatabase(ctx context.Context, db *sql.DB, dimensions int) error {

	cmds := []string{
		"INSTALL vss",
		"LOAD vss",
		fmt.Sprintf("CREATE TABLE embeddings(id TEXT PRIMARY KEY, content TEXT, vec FLOAT[%d])", dimensions),
		"CREATE INDEX idx ON embeddings USING HNSW (vec)",
	}

	for _, q := range cmds {

		slog.Debug(q)
		_, err := db.ExecContext(ctx, q)

		if err != nil {
			return fmt.Errorf("Failed to configure data - query failed, %w (%s)", err, q)
		}
	}

	return nil
}
