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

package github

import "net/http"

type TokenAuthStrategy struct {
	Token     string
	Transport http.RoundTripper
}

// RoundTrip implements the RoundTripper interface.
func (t *TokenAuthStrategy) RoundTrip(req *http.Request) (*http.Response, error) {
	// To set extra headers, we must make a copy of the Request so
	// that we don't modify the Request we were given. This is required by the
	// specification of http.RoundTripper.
	//
	// Since we are going to modify only req.Header here, we only need a deep copy
	// of req.Header.
	req2 := new(http.Request)
	*req2 = *req
	req2.Header = make(http.Header, len(req.Header))
	for k, s := range req.Header {
		req2.Header[k] = append([]string(nil), s...)
	}
	req2.Header.Add("Authorization", "token "+t.Token)

	return t.transport().RoundTrip(req2)
}

// AuthTransport implements the AuthStrategy interface.
func (t *TokenAuthStrategy) AuthTransport() (http.RoundTripper, error) {
	return t, nil
}

func (t *TokenAuthStrategy) transport() http.RoundTripper {
	if t.Transport != nil {
		return t.Transport
	}

	return http.DefaultTransport
}
