package bitbucketserver

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetWebhooksFollowsPagination(t *testing.T) {
	var requestedStarts []string

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/api/1.0/projects/PROJ/repos/repo/webhooks", request.URL.Path)

		start := request.URL.Query().Get("start")
		requestedStarts = append(requestedStarts, start)

		writer.Header().Set("Content-Type", "application/json")
		switch start {
		case "0":
			fmt.Fprint(writer, `{"size":1,"limit":500,"start":0,"isLastPage":false,"nextPageStart":1,
				"values":[{"id":1,"name":"other","url":"https://example.com/other"}]}`)
		default:
			fmt.Fprint(writer, `{"size":1,"limit":500,"start":1,"isLastPage":true,
				"values":[{"id":2,"name":"Argo Events","url":"https://example.com/argo-events"}]}`)
		}
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	client := &customBitbucketServerClient{
		client: server.Client(),
		ctx:    context.Background(),
		url:    serverURL,
	}

	webhooks, err := client.GetWebhooks("PROJ", "repo")
	require.NoError(t, err)
	require.Len(t, webhooks, 2)
	// A webhook that is only present on a later page must still be returned.
	assert.Equal(t, "https://example.com/argo-events", webhooks[1].Url)
	assert.Equal(t, []string{"0", "1"}, requestedStarts)
}

func TestGetWebhooksStopsOnNonAdvancingPage(t *testing.T) {
	requests := 0

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++

		writer.Header().Set("Content-Type", "application/json")
		// A response that keeps pointing at the current page must not loop forever.
		fmt.Fprint(writer, `{"size":1,"limit":500,"start":0,"isLastPage":false,"nextPageStart":0,
			"values":[{"id":1,"name":"Argo Events","url":"https://example.com/argo-events"}]}`)
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	client := &customBitbucketServerClient{
		client: server.Client(),
		ctx:    context.Background(),
		url:    serverURL,
	}

	webhooks, err := client.GetWebhooks("PROJ", "repo")
	require.NoError(t, err)
	assert.Len(t, webhooks, 1)
	assert.Equal(t, 1, requests)
}
