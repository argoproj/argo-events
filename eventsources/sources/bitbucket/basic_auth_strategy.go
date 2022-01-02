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
	bitbucketv2 "github.com/ktrysmt/go-bitbucket"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"

	"github.com/argoproj/argo-events/common"
)

type BasicAuthStrategy struct {
	username string
	password string
}

func NewBasicAuthStrategy(usernameSecret, passwordSecret *corev1.SecretKeySelector) (*BasicAuthStrategy, error) {
	username, err := common.GetSecretFromVolume(usernameSecret)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve bitbucket username from secret")
	}

	password, err := common.GetSecretFromVolume(passwordSecret)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve bitbucket password from secret")
	}

	return &BasicAuthStrategy{
		username: username,
		password: password,
	}, nil
}

// Client implements the AuthStrategy interface.
func (as *BasicAuthStrategy) BitbucketClient() *bitbucketv2.Client {
	return bitbucketv2.NewBasicAuth(as.username, as.password)
}
