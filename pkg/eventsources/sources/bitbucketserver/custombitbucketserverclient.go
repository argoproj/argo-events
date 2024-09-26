package bitbucketserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// customBitbucketServerClient returns a Bitbucket Server HTTP client that implements methods that gfleury/go-bitbucket-v1 does not.
// Specifically getting Pull Requests associated to a commit is not supported by gfleury/go-bitbucket-v1.
type customBitbucketServerClient struct {
	client *http.Client
	ctx    context.Context
	token  string
	url    *url.URL
}

// pullRequestRes is a struct containing information about the Pull Request.
type pullRequestRes struct {
	ID    int    `json:"id"`
	State string `json:"state"`
}

// pagedPullRequestsRes is a paged response with values of pullRequestRes.
type pagedPullRequestsRes struct {
	Size          int              `json:"size"`
	Limit         int              `json:"limit"`
	IsLastPage    bool             `json:"isLastPage"`
	Values        []pullRequestRes `json:"values"`
	Start         int              `json:"start"`
	NextPageStart int              `json:"nextPageStart"`
}

type pagination struct {
	Start int
	Limit int
}

func (p *pagination) StartStr() string {
	return strconv.Itoa(p.Start)
}

func (p *pagination) LimitStr() string {
	return strconv.Itoa(p.Limit)
}

func (c *customBitbucketServerClient) authHeader(req *http.Request) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
}

func (c *customBitbucketServerClient) get(u string) ([]byte, error) {
	req, err := http.NewRequestWithContext(c.ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	c.authHeader(req)

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode > 299 {
		resBody, readErr := io.ReadAll(res.Body)
		if readErr != nil {
			return nil, readErr
		}

		return nil, fmt.Errorf("calling endpoint '%s' failed: status %d: response body: %s", u, res.StatusCode, resBody)
	}

	return io.ReadAll(res.Body)
}

// GetCommitPullRequests returns all the Pull Requests associated to the commit id.
func (c *customBitbucketServerClient) GetCommitPullRequests(project, repository, commit string) ([]pullRequestRes, error) {
	p := pagination{Start: 0, Limit: 500}

	commitsURL := c.url.JoinPath(fmt.Sprintf("api/1.0/projects/%s/repos/%s/commits/%s/pull-requests", project, repository, commit))
	query := commitsURL.Query()
	query.Set("limit", p.LimitStr())

	var pullRequests []pullRequestRes
	for {
		query.Set("start", p.StartStr())
		commitsURL.RawQuery = query.Encode()

		body, err := c.get(commitsURL.String())
		if err != nil {
			return nil, err
		}

		var pagedPullRequests pagedPullRequestsRes
		err = json.Unmarshal(body, &pagedPullRequests)
		if err != nil {
			return nil, err
		}

		pullRequests = append(pullRequests, pagedPullRequests.Values...)

		if pagedPullRequests.IsLastPage {
			break
		}

		p.Start = pagedPullRequests.NextPageStart
	}

	return pullRequests, nil
}
