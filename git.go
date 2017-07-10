package borges

import (
	"gopkg.in/src-d/core-retrieval.v0/model"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

func NewGitReferencer(r *git.Repository) Referencer {
	return gitReferencer{r}
}

type gitReferencer struct {
	*git.Repository
}

func (r gitReferencer) References() ([]*model.Reference, error) {
	iter, err := r.Repository.References()
	if err != nil {
		return nil, err
	}

	var refs []*model.Reference
	return refs, iter.ForEach(func(ref *plumbing.Reference) error {
		//TODO: add tags support
		if ref.Type() != plumbing.HashReference || ref.IsRemote() {
			return nil
		}

		roots, err := rootCommits(r.Repository, plumbing.NewHash(ref.Hash().String()))
		if err != nil {
			return err
		}

		refs = append(refs, &model.Reference{
			Name:  ref.Name().String(),
			Hash:  model.NewSHA1(ref.Hash().String()),
			Init:  roots[0],
			Roots: roots,
		})
		return nil
	})
}

func rootCommits(r *git.Repository, from plumbing.Hash) ([]model.SHA1, error) {
	h, err := ResolveHash(r, from)
	if err != nil {
		return nil, err
	}

	var roots []model.SHA1

	cIter, err := r.Log(&git.LogOptions{From: h})
	if err != nil {
		return nil, err
	}

	err = cIter.ForEach(func(wc *object.Commit) error {
		if wc.NumParents() == 0 {
			roots = append(roots, model.SHA1(wc.Hash))
		}

		return nil
	})

	return roots, err
}
