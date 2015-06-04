/*
   Copyright (C) 2014  Salsita s.r.o.

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program. If not, see {http://www.gnu.org/licenses/}.
*/

package jira

import (
	// Stdlib
	"crypto/rsa"
	"net/http"
	"net/url"

	// Vendor
	"github.com/tchap/oauth"
)

func NewOAuthClient(
	jiraBaseURL *url.URL,
	consumerKey string,
	privateKey *rsa.PrivateKey,
	accessToken string,
) *http.Client {

	return &http.Client{
		Transport: newOAuthRoundTripper(jiraBaseURL, consumerKey, privateKey, accessToken),
	}
}

type oauthRoundTripper struct {
	consumer *oauth.Consumer
	token    *oauth.AccessToken
}

func newOAuthRoundTripper(
	jiraBaseURL *url.URL,
	consumerKey string,
	privateKey *rsa.PrivateKey,
	accessToken string,
) *oauthRoundTripper {

	requestTokenURL, _ := url.Parse("/plugins/servlet/oauth/request-token")
	authorizeTokenURL, _ := url.Parse("/plugins/servlet/oauth/authorize")
	accessTokenURL, _ := url.Parse("/plugins/servlet/oauth/access-token")

	provider := oauth.ServiceProvider{
		RequestTokenUrl:   jiraBaseURL.ResolveReference(requestTokenURL).String(),
		AuthorizeTokenUrl: jiraBaseURL.ResolveReference(authorizeTokenURL).String(),
		AccessTokenUrl:    jiraBaseURL.ResolveReference(accessTokenURL).String(),
		HttpMethod:        "POST",
	}

	return &oauthRoundTripper{
		consumer: oauth.NewRSAConsumer(consumerKey, privateKey, provider),
		token:    &oauth.AccessToken{Token: accessToken},
	}
}

func (rt *oauthRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return rt.consumer.MakeRequest(r, rt.token)
}
