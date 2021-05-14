package gh

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v35/github"
)

type Gh struct {
	client *github.Client
}

type PullRequestFile struct {
	Content *github.RepositoryContent
	Mode    string
	Type    string
}

func New() (*Gh, error) {
	// GITHUB_TOKEN
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("env %s is not set", "GITHUB_TOKEN")
	}
	v3c := github.NewClient(httpClient(token))
	if v3ep := os.Getenv("GITHUB_API_URL"); v3ep != "" {
		baseEndpoint, err := url.Parse(v3ep)
		if err != nil {
			return nil, err
		}
		if !strings.HasSuffix(baseEndpoint.Path, "/") {
			baseEndpoint.Path += "/"
		}
		v3c.BaseURL = baseEndpoint
	}

	return &Gh{
		client: v3c,
	}, nil
}

func Parse(in string) (string, string, int, error) {
	u, err := url.Parse(in)
	if err != nil {
		return "", "", 0, err
	}
	splitted := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	switch len(splitted) {
	case 0, 1:
		return "", "", 0, fmt.Errorf("could not parse: %s", in)
	case 2:
		owner := splitted[0]
		repo := splitted[1]
		return owner, repo, 0, nil
	case 3:
		return "", "", 0, fmt.Errorf("could not parse: %s", in)
	case 4:
		if splitted[2] != "pull" {
			return "", "", 0, fmt.Errorf("could not parse: %s", in)
		}
		number, err := strconv.Atoi(splitted[3])
		if err != nil {
			return "", "", 0, fmt.Errorf("could not parse: %s", in)
		}
		owner := splitted[0]
		repo := splitted[1]
		return owner, repo, number, nil
	default:
		return "", "", 0, fmt.Errorf("could not parse: %s", in)
	}
}

func (g *Gh) GetPullRequest(ctx context.Context, owner, repo string, number int) (*github.PullRequest, []*PullRequestFile, error) {
	pr, _, err := g.client.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return nil, nil, err
	}
	files, _, err := g.client.PullRequests.ListFiles(ctx, owner, repo, number, &github.ListOptions{
		PerPage: 1000,
	})
	if err != nil {
		return nil, nil, err
	}

	tree, _, err := g.client.Git.GetTree(ctx, owner, repo, pr.GetHead().GetSHA(), true)
	if err != nil {
		return nil, nil, err
	}
	entries := map[string]*PullRequestFile{}
	for _, e := range tree.Entries {
		entries[e.GetPath()] = &PullRequestFile{
			Mode: e.GetMode(),
			Type: e.GetType(),
		}
	}

	prFiles := []*PullRequestFile{}
	for _, f := range files {
		fc, _, _, err := g.client.Repositories.GetContents(ctx, owner, repo, f.GetFilename(), &github.RepositoryContentGetOptions{
			Ref: pr.Head.GetRef(),
		})
		if err != nil {
			return nil, nil, err
		}
		prFile, ok := entries[f.GetFilename()]
		if !ok {
			return nil, nil, fmt.Errorf("not found: %s", f.GetFilename())
		}
		prFile.Content = fc
		prFiles = append(prFiles, prFile)
	}

	return pr, prFiles, nil
}

func (g *Gh) GetRepository(ctx context.Context, owner, repo string) (*github.Repository, error) {
	r, _, err := g.client.Repositories.Get(ctx, owner, repo)
	return r, err
}

func (g *Gh) CopyPullRequest(ctx context.Context, owner, repo string, pr *github.PullRequest, files []*PullRequestFile) error {
	_, _ = fmt.Fprintf(os.Stderr, "Copying %s/%s pull request #%d to %s/%s ... ", pr.GetHead().GetUser().GetLogin(), pr.GetHead().GetRepo().GetName(), pr.GetNumber(), owner, repo)
	defer func() {
		_, _ = fmt.Fprintln(os.Stderr, "")
	}()
	r, _, err := g.client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return err
	}
	base := r.GetDefaultBranch()
	gitRef, err := g.createBranch(ctx, owner, repo, pr.GetHead().GetRef(), base)
	if err != nil {
		return err
	}
	parent, _, err := g.client.Git.GetCommit(ctx, owner, repo, gitRef.GetObject().GetSHA())
	if err != nil {
		return err
	}

	entries := []*github.TreeEntry{}
	for _, f := range files {
		blob, _, err := g.client.Git.CreateBlob(ctx, owner, repo, &github.Blob{
			Content:  f.Content.Content,
			Encoding: github.String(f.Content.GetEncoding()),
			Size:     github.Int(f.Content.GetSize()),
		})
		if err != nil {
			return err
		}
		entry := &github.TreeEntry{
			Path: github.String(f.Content.GetPath()),
			Mode: github.String(f.Mode),
			Type: github.String(f.Type),
			SHA:  github.String(blob.GetSHA()),
		}
		entries = append(entries, entry)
	}
	tree, _, err := g.client.Git.CreateTree(ctx, owner, repo, gitRef.GetObject().GetSHA(), entries)
	if err != nil {
		return err
	}

	commit, _, err := g.client.Git.CreateCommit(ctx, owner, repo, &github.Commit{
		Message: github.String(pr.GetTitle()),
		Tree:    tree,
		Parents: []*github.Commit{parent},
	})

	nref := &github.Reference{
		Ref: github.String(gitRef.GetRef()),
		Object: &github.GitObject{
			Type: github.String("commit"),
			SHA:  github.String(commit.GetSHA()),
		},
	}

	if _, _, err := g.client.Git.UpdateRef(ctx, owner, repo, nref, true); err != nil {
		return err
	}

	draft := true
	if r.GetVisibility() == "private" {
		draft = false
	}

	npr, _, err := g.client.PullRequests.Create(ctx, owner, repo, &github.NewPullRequest{
		Title: github.String(pr.GetTitle()),
		Head:  github.String(gitRef.GetRef()),
		Base:  github.String(base),
		Body:  github.String(pr.GetBody()),
		Draft: github.Bool(draft),
	})
	if err != nil {
		return err
	}
	if draft {
		_, _ = fmt.Fprintf(os.Stderr, "%s as draft", npr.GetHTMLURL())
	} else {
		_, _ = fmt.Fprintf(os.Stderr, "%s", npr.GetHTMLURL())
	}
	return nil
}

func (g *Gh) createBranch(ctx context.Context, owner, repo, head, base string) (*github.Reference, error) {
	baseRef := fmt.Sprintf("refs/heads/%s", base)
	baseGitRef, _, err := g.client.Git.GetRef(ctx, owner, repo, baseRef)
	if err != nil {
		return nil, err
	}
	ref := fmt.Sprintf("refs/heads/%s", head)
	url := strings.Replace(baseGitRef.GetURL(), baseRef, ref, 1)
	gitRef, _, err := g.client.Git.CreateRef(ctx, owner, repo, &github.Reference{
		Ref:    &ref,
		URL:    &url,
		Object: baseGitRef.GetObject(),
	})
	if err != nil {
		return nil, err
	}
	return gitRef, nil
}

type roundTripper struct {
	transport   *http.Transport
	accessToken string
}

func (rt roundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("Authorization", fmt.Sprintf("token %s", rt.accessToken))
	return rt.transport.RoundTrip(r)
}

func httpClient(token string) *http.Client {
	t := &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 5 * time.Second,
	}
	rt := roundTripper{
		transport:   t,
		accessToken: token,
	}
	return &http.Client{
		Timeout:   time.Second * 10,
		Transport: rt,
	}
}
