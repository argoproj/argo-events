package bitbucketserver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
	bitbucketv1 "github.com/gfleury/go-bitbucket-v1"
	"github.com/mitchellh/mapstructure"
)

func newBitbucketServerClientCfg(bitbucketserverEventSource *v1alpha1.BitbucketServerEventSource) (*bitbucketv1.Configuration, error) {
	bitbucketCfg := bitbucketv1.NewConfiguration(bitbucketserverEventSource.BitbucketServerBaseURL)
	bitbucketCfg.AddDefaultHeader("x-atlassian-token", "no-check")
	bitbucketCfg.AddDefaultHeader("x-requested-with", "XMLHttpRequest")
	bitbucketCfg.HTTPClient = &http.Client{}

	if bitbucketserverEventSource.TLS != nil {
		tlsConfig, err := sharedutil.GetTLSConfig(bitbucketserverEventSource.TLS)
		if err != nil {
			return nil, fmt.Errorf("failed to get the tls configuration: %w", err)
		}

		bitbucketCfg.HTTPClient.Transport = &http.Transport{
			TLSClientConfig: tlsConfig,
		}
	}

	return bitbucketCfg, nil
}

func newBitbucketServerClient(ctx context.Context, bitbucketConfig *bitbucketv1.Configuration, bitbucketToken string) *bitbucketv1.APIClient {
	ctx = context.WithValue(ctx, bitbucketv1.ContextAccessToken, bitbucketToken)
	return bitbucketv1.NewAPIClient(ctx, bitbucketConfig)
}

type bitbucketServerReposPager struct {
	Size          int                      `json:"size"`
	Limit         int                      `json:"limit"`
	Start         int                      `json:"start"`
	NextPageStart int                      `json:"nextPageStart"`
	IsLastPage    bool                     `json:"isLastPage"`
	Values        []bitbucketv1.Repository `json:"values"`
}

// getProjectRepositories returns all the Bitbucket Server repositories in the provided projects.
func getProjectRepositories(client *bitbucketv1.APIClient, projects []string) ([]v1alpha1.BitbucketServerRepository, error) {
	var bitbucketRepos []bitbucketv1.Repository
	for _, project := range projects {
		paginationOptions := map[string]interface{}{"start": 0, "limit": 500}
		for {
			response, err := client.DefaultApi.GetRepositoriesWithOptions(project, paginationOptions)
			if err != nil {
				return nil, fmt.Errorf("unable to list repositories for project %s: %w", project, err)
			}

			var reposPager bitbucketServerReposPager
			err = mapstructure.Decode(response.Values, &reposPager)
			if err != nil {
				return nil, fmt.Errorf("unable to decode repositories for project %s: %w", project, err)
			}

			bitbucketRepos = append(bitbucketRepos, reposPager.Values...)

			if reposPager.IsLastPage {
				break
			}

			paginationOptions["start"] = reposPager.NextPageStart
		}
	}

	var repositories []v1alpha1.BitbucketServerRepository
	for n := range bitbucketRepos {
		repositories = append(repositories, v1alpha1.BitbucketServerRepository{
			ProjectKey:     bitbucketRepos[n].Project.Key,
			RepositorySlug: bitbucketRepos[n].Slug,
		})
	}

	return repositories, nil
}

type refsChangedWebhookEvent struct {
	EventKey string `json:"eventKey"`
	Date     string `json:"date"`
	Actor    struct {
		Name         string `json:"name"`
		EmailAddress string `json:"emailAddress"`
		ID           int    `json:"id"`
		DisplayName  string `json:"displayName"`
		Active       bool   `json:"active"`
		Slug         string `json:"slug"`
		Type         string `json:"type"`
		Links        struct {
			Self []struct {
				Href string `json:"href"`
			} `json:"self"`
		} `json:"links"`
	} `json:"actor"`
	Repository struct {
		Slug          string `json:"slug"`
		ID            int    `json:"id"`
		Name          string `json:"name"`
		HierarchyID   string `json:"hierarchyId"`
		ScmID         string `json:"scmId"`
		State         string `json:"state"`
		StatusMessage string `json:"statusMessage"`
		Forkable      bool   `json:"forkable"`
		Project       struct {
			Key    string `json:"key"`
			ID     int    `json:"id"`
			Name   string `json:"name"`
			Public bool   `json:"public"`
			Type   string `json:"type"`
			Links  struct {
				Self []struct {
					Href string `json:"href"`
				} `json:"self"`
			} `json:"links"`
		} `json:"project"`
		Public bool `json:"public"`
		Links  struct {
			Clone []struct {
				Href string `json:"href"`
				Name string `json:"name"`
			} `json:"clone"`
			Self []struct {
				Href string `json:"href"`
			} `json:"self"`
		} `json:"links"`
	} `json:"repository"`
	Changes []refsChangedWebHookEventChange `json:"changes"`
}

type refsChangedWebHookEventChange struct {
	Ref struct {
		ID        string `json:"id"`
		DisplayID string `json:"displayId"`
		Type      string `json:"type"`
	} `json:"ref"`
	RefID    string `json:"refId"`
	FromHash string `json:"fromHash"`
	ToHash   string `json:"toHash"`
	Type     string `json:"type"`
}
