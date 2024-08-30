package vector

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/aaronland/go-roster"
)

type Image struct {
	ID         string    `json:"id"`
	Embeddings []float32 `json:"embeddings"`
}

type Match struct {
	ID         string  `json:"id"`
	Similarity float32 `json:"similarity"`
}

// Database defines an interface for adding and querying vector embeddings of `Image` records.
type Database interface {
	// Add adds an `Image` record to the underlying database implementation.
	Add(context.Context, *Image) error
	// Query results a list of `Match` instances for images matching a `Image` in the underlying database implementation.
	Query(context.Context, *Image) ([]*Match, error)
	// MeetsThreshold returns a boolean value indicating whether a `QueryResult` instance satisfies a given threshold value.
	MeetsThreshold(context.Context, *Match, float64) (bool, error)
	// Close performs and terminating functions required by the database.
	Close(context.Context) error
}

// DatabaseInitializationFunc is a function defined by individual database package and used to create
// an instance of that database
type DatabaseInitializationFunc func(ctx context.Context, uri string) (Database, error)

var database_roster roster.Roster

// RegisterDatabase registers 'scheme' as a key pointing to 'init_func' in an internal lookup table
// used to create new `Database` instances by the `NewDatabase` method.
func RegisterDatabase(ctx context.Context, scheme string, init_func DatabaseInitializationFunc) error {

	err := ensureDatabaseRoster()

	if err != nil {
		return err
	}

	return database_roster.Register(ctx, scheme, init_func)
}

func ensureDatabaseRoster() error {

	if database_roster == nil {

		r, err := roster.NewDefaultRoster()

		if err != nil {
			return err
		}

		database_roster = r
	}

	return nil
}

// NewDatabase returns a new `Database` instance configured by 'uri'. The value of 'uri' is parsed
// as a `url.URL` and its scheme is used as the key for a corresponding `DatabaseInitializationFunc`
// function used to instantiate the new `Database`. It is assumed that the scheme (and initialization
// function) have been registered by the `RegisterDatabase` method.
func NewDatabase(ctx context.Context, uri string) (Database, error) {

	u, err := url.Parse(uri)

	if err != nil {
		return nil, err
	}

	scheme := u.Scheme

	i, err := database_roster.Driver(ctx, scheme)

	if err != nil {
		return nil, err
	}

	init_func := i.(DatabaseInitializationFunc)
	return init_func(ctx, uri)
}

// Schemes returns the list of schemes that have been registered.
func DatabaseSchemes() []string {

	ctx := context.Background()
	schemes := []string{}

	err := ensureDatabaseRoster()

	if err != nil {
		return schemes
	}

	for _, dr := range database_roster.Drivers(ctx) {
		scheme := fmt.Sprintf("%s://", strings.ToLower(dr))
		schemes = append(schemes, scheme)
	}

	sort.Strings(schemes)
	return schemes
}
