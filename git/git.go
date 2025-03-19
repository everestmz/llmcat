package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// FindRepoRoot takes a filepath (relative or absolute) and checks if it's within a Git repository.
// If found, it returns the absolute path to the repository root and true; otherwise, an empty string and false.
func FindRepoRoot(path string) (string, bool) {
	// Convert to absolute path if relative
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", false
	}

	// Check if the path exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return "", false
	}

	// Start from the given path and walk up the directory tree
	currentPath := absPath
	for {
		// Try to open a Git repository at the current path
		repo, err := git.PlainOpen(currentPath)
		if err == nil {
			// Successfully opened a repo; get its worktree root
			wt, err := repo.Worktree()
			if err != nil {
				return "", false
			}
			root := wt.Filesystem.Root()
			return root, true
		}

		// If no repo found, move up one directory
		parent := filepath.Dir(currentPath)
		if parent == currentPath {
			// Reached the filesystem root (e.g., "/" or "C:\") without finding a repo
			return "", false
		}
		currentPath = parent
	}
}

func NewRepo(path string) (*Repo, error) {
	_, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	root, ok := FindRepoRoot(path)
	if !ok {
		return nil, fmt.Errorf("Unable to find any git repo in %s or parent directories", path)
	}

	repo, err := git.PlainOpen(root)
	if err != nil {
		return nil, err
	}

	return &Repo{
		repo:     repo,
		repoRoot: root,
	}, nil
}

type Repo struct {
	repo     *git.Repository
	repoRoot string
}

func (r *Repo) Status() (git.Status, error) {
	worktree, err := r.repo.Worktree()
	if err != nil {
		return nil, err
	}

	return worktree.Status()
}

type LsFilesOptions struct {
	IncludeUntrackedFiles bool
}

type File struct {
	Name string
	Mode filemode.FileMode
}

func (r *Repo) LsFiles(path string, options *LsFilesOptions) ([]string, error) {
	var files []string

	err := r.LsFilesFunc(path, func(f *File) error {
		files = append(files, f.Name)
		return nil
	}, options)

	return files, err
}

func (r *Repo) LsFilesFunc(path string, fn func(f *File) error, options *LsFilesOptions) error {
	head, err := r.repo.Head()
	if err != nil {
		return err
	}

	commit, err := r.repo.CommitObject(head.Hash())
	if err != nil {
		return err
	}

	tree, err := commit.Tree()
	if err != nil {
		return err
	}

	path = filepath.Clean(path)
	if filepath.IsAbs(path) {
		path, err = filepath.Rel(r.repoRoot, path)
		if err != nil {
			return err
		}
	}

	// Git doesn't like these
	path = strings.TrimPrefix(path, "./")
	path = strings.TrimPrefix(path, ".")

	if path != "" {
		tree, err = tree.Tree(path)
		if err != nil {
			return err
		}
	}

	err = tree.Files().ForEach(func(f *object.File) error {
		return fn(&File{
			Name: filepath.Join(path, f.Name),
			Mode: f.Mode,
		})
	})
	if err != nil {
		return err
	}

	if options.IncludeUntrackedFiles {
		status, err := r.Status()
		if err != nil {
			return err
		}

		for path, info := range status {
			if info.Staging == git.Untracked {
				f, err := os.Open(path)
				if err != nil {
					return err
				}

				info, err := f.Stat()
				if err != nil {
					return err
				}

				err = fn(&File{
					Name: f.Name(),
					Mode: filemode.FileMode(info.Mode()),
				})
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
