package server

import (
	"io/ioutil"

	"golang.org/x/oauth2"

	"github.com/juju/errors"
	yaml "gopkg.in/yaml.v2"
)

type OAuthCreds struct {
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
}

type clientIDGetResp struct {
	ClientID string `json:"clientID"`
}

var googleEndpoint = oauth2.Endpoint{
	AuthURL:  "https://accounts.google.com/o/oauth2/auth",
	TokenURL: "https://accounts.google.com/o/oauth2/token",
}

func ReadOAuthCredsFile(credsFile string) (*OAuthCreds, error) {
	creds := &OAuthCreds{}
	contents, err := ioutil.ReadFile(credsFile)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if err := yaml.Unmarshal(contents, creds); err != nil {
		return nil, errors.Annotatef(err, "unmarshalling OAuth creds")
	}
	if creds.ClientID == "" || creds.ClientSecret == "" {
		return nil, errors.Errorf("%s: client_id and client_secret are required", credsFile)
	}
	return creds, nil
}
