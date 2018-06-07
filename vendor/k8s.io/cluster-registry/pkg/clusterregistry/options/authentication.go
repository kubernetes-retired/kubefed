/*
Copyright 2017 The Kubernetes Authors.

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

package options

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/spf13/pflag"

	genericapiserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	"k8s.io/cluster-registry/pkg/clusterregistry/authenticator"
)

type StandaloneAuthenticationOptions struct {
	Anonymous    *AnonymousAuthenticationOptions
	ClientCert   *genericoptions.ClientCertAuthenticationOptions
	PasswordFile *PasswordFileAuthenticationOptions
	TokenFile    *TokenFileAuthenticationOptions
	WebHook      *WebHookAuthenticationOptions

	TokenSuccessCacheTTL time.Duration
	TokenFailureCacheTTL time.Duration
}

type AnonymousAuthenticationOptions struct {
	Allow bool
}

type PasswordFileAuthenticationOptions struct {
	BasicAuthFile string
}

type TokenFileAuthenticationOptions struct {
	TokenFile string
}

type WebHookAuthenticationOptions struct {
	ConfigFile string
	CacheTTL   time.Duration
}

func NewStandaloneAuthenticationOptions() *StandaloneAuthenticationOptions {
	return &StandaloneAuthenticationOptions{
		TokenSuccessCacheTTL: 10 * time.Second,
		TokenFailureCacheTTL: 0 * time.Second,
	}
}

func (s *StandaloneAuthenticationOptions) WithAll() *StandaloneAuthenticationOptions {
	return s.
		WithAnonymous().
		WithClientCert().
		WithPasswordFile().
		WithTokenFile().
		WithWebHook()
}

func (s *StandaloneAuthenticationOptions) WithAnonymous() *StandaloneAuthenticationOptions {
	s.Anonymous = &AnonymousAuthenticationOptions{Allow: true}
	return s
}

func (s *StandaloneAuthenticationOptions) WithClientCert() *StandaloneAuthenticationOptions {
	s.ClientCert = &genericoptions.ClientCertAuthenticationOptions{}
	return s
}

func (s *StandaloneAuthenticationOptions) WithPasswordFile() *StandaloneAuthenticationOptions {
	s.PasswordFile = &PasswordFileAuthenticationOptions{}
	return s
}

func (s *StandaloneAuthenticationOptions) WithTokenFile() *StandaloneAuthenticationOptions {
	s.TokenFile = &TokenFileAuthenticationOptions{}
	return s
}

func (s *StandaloneAuthenticationOptions) WithWebHook() *StandaloneAuthenticationOptions {
	s.WebHook = &WebHookAuthenticationOptions{
		CacheTTL: 2 * time.Minute,
	}
	return s
}

// Validate checks invalid config combination
func (s *StandaloneAuthenticationOptions) Validate() []error {
	return []error{}
}

func (s *StandaloneAuthenticationOptions) AddFlags(fs *pflag.FlagSet) {
	if s.Anonymous != nil {
		fs.BoolVar(&s.Anonymous.Allow, "anonymous-auth", s.Anonymous.Allow, ""+
			"Enables anonymous requests to the secure port of the API server. "+
			"Requests that are not rejected by another authentication method are treated as anonymous requests. "+
			"Anonymous requests have a username of system:anonymous, and a group name of system:unauthenticated.")
	}

	if s.ClientCert != nil {
		s.ClientCert.AddFlags(fs)
	}

	if s.PasswordFile != nil {
		fs.StringVar(&s.PasswordFile.BasicAuthFile, "basic-auth-file", s.PasswordFile.BasicAuthFile, ""+
			"If set, the file that will be used to admit requests to the secure port of the API server "+
			"via http basic authentication.")
	}

	if s.TokenFile != nil {
		fs.StringVar(&s.TokenFile.TokenFile, "token-auth-file", s.TokenFile.TokenFile, ""+
			"If set, the file that will be used to secure the secure port of the API server "+
			"via token authentication.")
	}

	if s.WebHook != nil {
		fs.StringVar(&s.WebHook.ConfigFile, "authentication-token-webhook-config-file", s.WebHook.ConfigFile, ""+
			"File with webhook configuration for token authentication in kubeconfig format. "+
			"The API server will query the remote service to determine authentication for bearer tokens.")

		fs.DurationVar(&s.WebHook.CacheTTL, "authentication-token-webhook-cache-ttl", s.WebHook.CacheTTL,
			"The duration to cache responses from the webhook token authenticator.")
	}
}

func (o *StandaloneAuthenticationOptions) ApplyTo(c *genericapiserver.Config) error {
	var err error
	if o.ClientCert != nil {
		c, err = c.ApplyClientCert(o.ClientCert.ClientCA)
		if err != nil {
			return fmt.Errorf("unable to load client CA file: %v", err)
		}
	}

	c.SupportsBasicAuth = o.PasswordFile != nil && len(o.PasswordFile.BasicAuthFile) > 0

	authenticator, securityDefinitions, err := o.toAuthenticationConfig().New()
	if err != nil {
		return err
	}
	c.Authenticator = authenticator
	if c.OpenAPIConfig != nil {
		c.OpenAPIConfig.SecurityDefinitions = securityDefinitions
	}
	return nil
}

func (s *StandaloneAuthenticationOptions) toAuthenticationConfig() authenticator.Config {
	ret := authenticator.Config{
		TokenSuccessCacheTTL: s.TokenSuccessCacheTTL,
		TokenFailureCacheTTL: s.TokenFailureCacheTTL,
	}

	if s.Anonymous != nil {
		ret.Anonymous = s.Anonymous.Allow
	}

	if s.ClientCert != nil {
		ret.ClientCAFile = s.ClientCert.ClientCA
	}

	if s.PasswordFile != nil {
		ret.BasicAuthFile = s.PasswordFile.BasicAuthFile
	}

	if s.TokenFile != nil {
		ret.TokenAuthFile = s.TokenFile.TokenFile
	}

	if s.WebHook != nil {
		ret.WebhookTokenAuthnConfigFile = s.WebHook.ConfigFile
		ret.WebhookTokenAuthnCacheTTL = s.WebHook.CacheTTL

		if len(s.WebHook.ConfigFile) > 0 && s.WebHook.CacheTTL > 0 {
			if s.TokenSuccessCacheTTL > 0 && s.WebHook.CacheTTL < s.TokenSuccessCacheTTL {
				glog.Warningf("the webhook cache ttl of %s is shorter than the overall cache ttl of %s for successful token authentication attempts.", s.WebHook.CacheTTL, s.TokenSuccessCacheTTL)
			}
			if s.TokenFailureCacheTTL > 0 && s.WebHook.CacheTTL < s.TokenFailureCacheTTL {
				glog.Warningf("the webhook cache ttl of %s is shorter than the overall cache ttl of %s for failed token authentication attempts.", s.WebHook.CacheTTL, s.TokenFailureCacheTTL)
			}
		}
	}

	return ret
}
