/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package bitbucket

import (
	"github.com/ktrysmt/go-bitbucket"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"

	"github.com/argoproj/argo-events/common"
)

type OAuthTokenAuthStrategy struct {
	token string
}

func NewOAuthTokenAuthStrategy(oauthTokenSecret *corev1.SecretKeySelector) (*OAuthTokenAuthStrategy, error) {
	token, err := common.GetSecretFromVolume(oauthTokenSecret)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve bitbucket oauth token from secret")
	}

	return &OAuthTokenAuthStrategy{
		token: token,
	}, nil
}

// Client implements the AuthStrategy interface.
func (as *OAuthTokenAuthStrategy) BitbucketClient() *bitbucket.Client {
	return bitbucket.NewOAuthbearerToken(as.token)
}
