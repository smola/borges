package borges

import (
	stderrors "errors"
	"fmt"
	"io"
	"strings"

	"github.com/inconshreveable/log15"
	"github.com/satori/go.uuid"
	"gopkg.in/src-d/core-retrieval.v0"
	"gopkg.in/src-d/core-retrieval.v0/model"
	"gopkg.in/src-d/go-errors.v0"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-kallax.v1"
)

var (
	log = log15.New()

	// ErrAlreadyStopped signals that an operation cannot be done because
	// the entity is already sopped.
	ErrAlreadyStopped = errors.NewKind("already stopped: %s")

	ErrWaitForJobs = errors.NewKind("no more jobs at the moment")

	ErrReferencedObjectTypeNotSupported error = stderrors.New("referenced object type not supported")
)

// Job represents a borges job to fetch and archive a repository.
type Job struct {
	RepositoryID uuid.UUID
}

// JobIter is an iterator of Job.
type JobIter interface {
	io.Closer
	// Next returns the next job. It returns io.EOF if there are no more
	// jobs. If there are no more jobs at the moment, but there can be
	// in the future, it returns an error of kind ErrWaitForJobs.
	Next() (*Job, error)
}

// RepositoryID tries to find a repository by the endpoint into the database.
// If no repository is found, it creates a new one and returns the ID.
func RepositoryID(endpoint string, storer *model.RepositoryStore) (uuid.UUID, error) {
	rs, err := storer.Find(
		model.NewRepositoryQuery().
			Where(kallax.And(kallax.ArrayContains(
				model.Schema.Repository.Endpoints, endpoint,
			))),
	)
	if err != nil {
		return uuid.Nil, err
	}

	repositories, err := rs.All()
	if err != nil {
		return uuid.Nil, err
	}

	l := len(repositories)
	switch {
	case l == 0:
		r := model.NewRepository()
		r.Endpoints = []string{endpoint}
		if _, err := storer.Save(r); err != nil {
			return uuid.Nil, err
		}

		return uuid.UUID(r.ID), nil
	case l > 1:
		// TODO log error printing the ids and the endpoint
	}

	return uuid.UUID(repositories[0].ID), nil
}

// ResolveHash gets the hash of a commit that is referenced by a tag, per example.
// The only resolvable objects are Tags and Commits. If the object is not one of them,
// This method will return an ErrReferencedObjectTypeNotSupported. The output hash
// always will be a Commit hash.
func ResolveHash(r *git.Repository, h plumbing.Hash) (plumbing.Hash, error) {
	obj, err := r.Object(plumbing.AnyObject, h)
	if err != nil {
		return plumbing.ZeroHash, err
	}

	switch o := obj.(type) {
	case *object.Commit:
		return o.Hash, nil
	case *object.Tag:
		return ResolveHash(r, o.Target)
	default:
		log.Warn("referenced object not supported", "type", o.Type())
		return plumbing.ZeroHash, ErrReferencedObjectTypeNotSupported
	}
}

// TODO temporal
func DropTables(names ...string) {
	smt := fmt.Sprintf("DROP TABLE IF EXISTS %s;", strings.Join(names, ", "))
	if _, err := core.Database().Exec(smt); err != nil {
		panic(err)
	}
}

// TODO temporal
func DropIndexes(names ...string) {
	smt := fmt.Sprintf("DROP INDEX IF EXISTS %s;", strings.Join(names, ", "))
	if _, err := core.Database().Exec(smt); err != nil {
		panic(err)
	}
}

// TODO temporal delete when kallax implements it
func CreateRepositoryTable() {
	_, err := core.Database().Exec(`CREATE TABLE IF NOT EXISTS repositories (
	id uuid PRIMARY KEY,
	created_at timestamptz,
	updated_at timestamptz,
	endpoints text[],
	status varchar(20),
	fetched_at timestamptz,
	fetch_error_at timestamptz,
	last_commit_at timestamptz,
	_references jsonb
	);
	CREATE INDEX idx_endpoints on "repositories" USING GIN ("endpoints");`)

	if err != nil {
		panic(err)
	}
}

type RepositoryStorage interface {
	Repository(ic model.SHA1) *git.Repository
}

type Referencer interface {
	References() ([]*model.Reference, error)
}
