package borges

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/fixtures"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

func TestArchiverSuite(t *testing.T) {
	suite.Run(t, new(ArchiverSuite))
}

type ArchiverSuite struct {
	suite.Suite
	tmpDir string
	j      *Job

	lastCommit plumbing.Hash
}

func (s *ArchiverSuite) SetupSuite() {
	assert := assert.New(s.T())
	fixtures.Init()

	s.tmpDir = filepath.Join(os.TempDir(), "test_data")
	err := os.RemoveAll(s.tmpDir)
	assert.NoError(err)

	s.lastCommit = plumbing.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e5")

	s.j = &Job{
		URL: fmt.Sprintf("file://%s", fixtures.Basic().One().DotGit().Base()),
	}
}

func (s *ArchiverSuite) TearDownSuite() {
	assert := assert.New(s.T())

	err := fixtures.Clean()
	assert.NoError(err)

	err = os.RemoveAll(s.tmpDir)
	assert.NoError(err)
}

func (s *ArchiverSuite) TestCreateLocalRepository() {
	assert := assert.New(s.T())

	repo, err := createLocalRepository(s.tmpDir, s.j, []*Reference{
		{
			Hash: NewSHA1("918c48b83bd081e863dbe1b80f8998f058cd8294"),
			Name: "refs/remotes/origin/master",
			Init: NewSHA1("b029517f6300c2da0f4b651b8642506cd6aaf45d"),
		}, {
			// branch is up to date
			Hash: NewSHA1("e8d3ffab552895c19b9fcf7aa264d277cde33881"),
			Name: "refs/remotes/origin/branch",
			Init: NewSHA1("b029517f6300c2da0f4b651b8642506cd6aaf45d"),
		},
	})
	assert.Nil(err)

	c, err := repo.Commit(s.lastCommit)
	assert.Nil(c)
	assert.Error(err)

	err = repo.Fetch(&git.FetchOptions{})
	assert.NoError(err)

	c, err = repo.Commit(s.lastCommit)
	assert.NoError(err)
	assert.NotNil(c)

	iter, err := repo.Objects()
	assert.NoError(err)
	assert.NotNil(iter)

	count := 0
	iter.ForEach(func(o object.Object) error {
		count++
		return nil
	})

	// 1- last commit into master
	// 2,3 - trees
	// 4 - file added into commit
	assert.Equal(4, count)
}