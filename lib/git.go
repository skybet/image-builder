// Copyright (c) 2017 Adam Pointer

package lib

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	log "github.com/Sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	gitssh "gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
	"gopkg.in/src-d/go-git.v4/storage/memory"
)

type GitClient struct {
	url    *string
	auth   transport.AuthMethod
	repo   *git.Repository
	ref    *plumbing.Reference
	commit *object.Commit
}

// Git creates a new GitClient
func Git() *GitClient {
	return new(GitClient)
}

// SetUrl sets the git url to clone from
func (g *GitClient) SetUrl(u string) {
	g.url = &u
}

// SetKey takes a path to a private key used to authenticate with the remote
func (g *GitClient) SetKey(path string) error {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("Cannot read %s: %s", path, err)
	}

	signer, err := ssh.ParsePrivateKey(b)
	if err != nil {
		return fmt.Errorf("Cannot parse %s: %s", path, err)
	}
	g.auth = &gitssh.PublicKeys{User: "git", Signer: signer}
	return nil
}

// Clone does a single branch clone of a specific branch into memory
func (g *GitClient) Clone(b string) error {
	branch := fmt.Sprintf("refs/heads/%s", b)
	log.Infof("Cloning %s (%s)", *g.url, branch)

	r, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL:           *g.url,
		Auth:          g.auth,
		ReferenceName: plumbing.ReferenceName(branch),
		SingleBranch:  true,
	})
	if err != nil {
		return err
	}
	g.repo = r

	ref, err := g.repo.Head()
	if err != nil {
		return err
	}
	g.ref = ref

	commit, err := g.repo.CommitObject(g.ref.Hash())
	if err != nil {
		return err
	}
	g.commit = commit
	return nil
}

// FileChanged gets a slice of paths which have changed between HEAD and it's parent commit(s)
func (g *GitClient) FilesChanged() (*object.Changes, error) {
	var changes object.Changes

	prnts := g.commit.Parents()
	from, err := g.commit.Tree()
	if err != nil {
		return &changes, err
	}
	err = prnts.ForEach(func(p *object.Commit) error {
		to, err := p.Tree()
		if err != nil {
			return err
		}
		chgs, err := from.Diff(to)
		changes = append(changes, chgs...)

		if err != nil {
			return err
		}
		return nil
	})
	return &changes, err
}

// DirsChanged takes a slice of file paths and returns a slice of the directories containing those files and also the parents
// of each to account for when a file is changed in a subdir of a repository
func (g *GitClient) DirsChanged() (dirs []string, err error) {
	changes, err := g.FilesChanged()
	if err != nil {
		return
	}
	for _, c := range *changes {
		var d string
		if c.From.Name != "" {
			d = path.Dir(c.From.Name)
			dirs = append(dirs, d)
			dirs = append(dirs, g.parents(d, []string{})...)
		}

		if c.To.Name != "" {
			d = path.Dir(c.To.Name)
			dirs = append(dirs, d)
			dirs = append(dirs, g.parents(d, []string{})...)
		}
	}
	return Dedup(dirs), err
}

func (g *GitClient) parents(p string, state []string) []string {
	if p == string(os.PathSeparator) {
		return state
	}

	parts := strings.Split(p, string(os.PathSeparator))
	parent := strings.Join(parts[0:(len(parts)-1)], string(os.PathSeparator))
	state = append(state, parent)
	//return g.parents(parent, state)
	return state
}

// PathHasDockerfile determines if there is there a file called Dockerfile at the specified path
func (g *GitClient) PathHasDockerfile(filepath string) (yes bool, err error) {
	t, err := g.commit.Tree()
	if err != nil {
		return
	}

	_, err = t.File(path.Join(filepath, "Dockerfile"))
	if err == object.ErrFileNotFound {
		log.Debugf("No Dockerfile found at path: %s", filepath)
		return false, nil
	} else if err != nil {
		return
	}
	log.Debugf("Found Dockerfile at path: %s", filepath)
	return true, nil
}

// RemoveNoBuildPaths takes a slice of directories and removes any not containing a Dockerfile
func (g *GitClient) RemoveNonBuildPaths(paths []string) (roots []string, err error) {
	for _, value := range paths {

		log.Debugf("Checking path: %s", value)
		yes, err := g.PathHasDockerfile(value)
		if err != nil {
			return roots, err
		}
		if yes {
			roots = append(roots, value)
		}
	}
	return
}

func (g *GitClient) GetTarAtPath(dirpath string) (*bytes.Buffer, error) {
	b := new(bytes.Buffer)
	tw := tar.NewWriter(b)
	defer tw.Close()

	ta, err := g.commit.Tree()
	if err != nil {
		return b, err
	}
	tb, err := ta.Tree(dirpath)
	if err != nil {
		return b, err
	}
	tb.Files().ForEach(func(f *object.File) error {
		if err := g.addFile(tw, f); err != nil {
			return err
		}
		return nil
	})
	if err := tw.Close(); err != nil {
		return b, err
	}
	log.Debugf("Tar archive is %d bytes long", b.Len())
	return b, err
}

func (g *GitClient) addFile(tw *tar.Writer, f *object.File) (err error) {
	log.Debugf("Adding %s to tar archive", f.Name)
	// Write the header to the tarball archive
	header := new(tar.Header)
	header.Name = f.Name
	header.Size = f.Size
	header.Mode = int64(f.Mode)
	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	// Copy the file data to the tarball
	reader, err := f.Reader()
	if err != nil {
		return err
	}
	defer reader.Close()
	if _, err := io.Copy(tw, reader); err != nil {
		return err
	}
	return
}

func (g *GitClient) GenerateTags(path string) []string {
	hash := g.ref.Hash().String()
	return []string{
		fmt.Sprintf("%s:%s", path, g.ref.Name().Short()),
		fmt.Sprintf("%s:%s", path, hash[0:7]),
	}
}
