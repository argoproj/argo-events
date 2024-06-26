package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
)

func TestGetReplicas(t *testing.T) {
	ep := EventSourceSpec{}
	assert.Equal(t, ep.GetReplicas(), int32(1))
	ep.Replicas = convertInt(t, 0)
	assert.Equal(t, ep.GetReplicas(), int32(1))
	ep.Replicas = convertInt(t, 2)
	assert.Equal(t, ep.GetReplicas(), int32(2))
}

func convertInt(t *testing.T, num int) *int32 {
	t.Helper()
	r := int32(num)
	return &r
}

func TestGithubEventSourceGetRepositories(t *testing.T) {
	es := GithubEventSource{
		DeprecatedOwner:      "test",
		DeprecatedRepository: "test",
	}
	assert.Equal(t, len(es.GetOwnedRepositories()), 1)
	es = GithubEventSource{
		Repositories: []OwnedRepositories{
			{
				Owner: "test1",
				Names: []string{"t1", "t2"},
			},
			{
				Owner: "test2",
				Names: []string{"t3", "t4"},
			},
		},
	}
	assert.Equal(t, len(es.GetOwnedRepositories()), 2)
}

// TestShouldCreateWebhooks tests the ShouldCreateWebhooks function on the BitbucketServerEventSource type.
func TestShouldCreateWebhooks(t *testing.T) {
	// Create a dummy SecretKeySelector for testing
	dummySecretKeySelector := &v1.SecretKeySelector{
		LocalObjectReference: v1.LocalObjectReference{
			Name: "dummy-secret",
		},
		Key: "token",
	}

	tests := []struct {
		name string
		b    BitbucketServerEventSource
		want bool
	}{
		{
			name: "Valid Webhooks - Both Projects and Repositories",
			b: BitbucketServerEventSource{
				AccessToken:  dummySecretKeySelector,
				Webhook:      &WebhookContext{URL: "http://example.com"},
				Projects:     []string{"Project1"},
				Repositories: []BitbucketServerRepository{{ProjectKey: "Proj1", RepositorySlug: "Repo1"}},
			},
			want: true,
		},
		{
			name: "No AccessToken",
			b: BitbucketServerEventSource{
				Webhook:      &WebhookContext{URL: "http://example.com"},
				Projects:     []string{"Project1"},
				Repositories: []BitbucketServerRepository{{ProjectKey: "Proj1", RepositorySlug: "Repo1"}},
			},
			want: false,
		},
		{
			name: "No Webhook",
			b: BitbucketServerEventSource{
				AccessToken:  dummySecretKeySelector,
				Projects:     []string{"Project1"},
				Repositories: []BitbucketServerRepository{{ProjectKey: "Proj1", RepositorySlug: "Repo1"}},
			},
			want: false,
		},
		{
			name: "No URL",
			b: BitbucketServerEventSource{
				AccessToken:  dummySecretKeySelector,
				Webhook:      &WebhookContext{},
				Projects:     []string{"Project1"},
				Repositories: []BitbucketServerRepository{{ProjectKey: "Proj1", RepositorySlug: "Repo1"}},
			},
			want: false,
		},
		{
			name: "No Projects or Repositories",
			b: BitbucketServerEventSource{
				AccessToken: dummySecretKeySelector,
				Webhook:     &WebhookContext{URL: "http://example.com"},
			},
			want: false,
		},
		{
			name: "Valid with Only Repositories",
			b: BitbucketServerEventSource{
				AccessToken:  dummySecretKeySelector,
				Webhook:      &WebhookContext{URL: "http://example.com"},
				Repositories: []BitbucketServerRepository{{ProjectKey: "Proj1", RepositorySlug: "Repo1"}},
			},
			want: true,
		},
		{
			name: "Valid with Only Projects",
			b: BitbucketServerEventSource{
				AccessToken: dummySecretKeySelector,
				Webhook:     &WebhookContext{URL: "http://example.com"},
				Projects:    []string{"Project1"},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.b.ShouldCreateWebhooks(); got != tt.want {
				t.Errorf("%s: ShouldCreateWebhooks() = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
