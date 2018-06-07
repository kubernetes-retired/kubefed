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
	"errors"
	"time"

	"github.com/golang/glog"
	"github.com/spf13/pflag"

	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/apiserver/pkg/authorization/authorizerfactory"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/plugin/pkg/authorizer/webhook"
)

// StandaloneAuthorizerConfig configures an authorizer for the standalone
// cluster registry.
type StandaloneAuthorizerConfig struct {
	// AlwaysAllow allows all requests that are validated by an authenticator full
	// access to cluster registry resources. This overrides all other authorizers.
	AlwaysAllow bool

	// Webhook uses an external webhook to authorize requests.
	Webhook bool

	// WebhookConfigFile is the file that contains the webhook configuration in
	// kubeconfig format.
	WebhookConfigFile string

	// WebhookCacheAuthorizedTTL is the length of time that a successful webhook
	// authorization response will be cached.
	WebhookCacheAuthorizedTTL time.Duration

	// WebhookCacheUnauthorizedTTL is the length of time that an unsuccessful
	// authorization response will be cached. You generally want more responsive,
	// "deny, try again" flows.
	WebhookCacheUnauthorizedTTL time.Duration
}

// New creates a new authorizer based on the receiver's configured values.
func (s StandaloneAuthorizerConfig) New() (authorizer.Authorizer, error) {
	if s.AlwaysAllow {
		return authorizerfactory.NewAlwaysAllowAuthorizer(), nil
	}
	if s.Webhook {
		if s.WebhookConfigFile == "" {
			return nil, errors.New("webhook configuration file not provided")
		}
		return webhook.New(s.WebhookConfigFile, s.WebhookCacheAuthorizedTTL, s.WebhookCacheUnauthorizedTTL)
	}
	glog.Info("No authorizer specified; defaulting to AlwaysDeny")
	return authorizerfactory.NewAlwaysDenyAuthorizer(), nil
}

// StandaloneAuthorizationOptions configure authorization in the cluster registry
// when it is run as a standalone API server.
type StandaloneAuthorizationOptions struct {
	// AlwaysAllow allows all requests that are validated by an authenticator full
	// access to cluster registry resources. This overrides all other authorizers.
	AlwaysAllow bool

	// Webhook uses an external webhook to authorize requests.
	Webhook bool

	// WebhookConfigFile is the file that contains the webhook configuration in
	// kubeconfig format.
	WebhookConfigFile string

	// WebhookCacheAuthorizedTTL is the length of time that a successful webhook
	// authorization response will be cached.
	WebhookCacheAuthorizedTTL time.Duration

	// WebhookCacheUnauthorizedTTL is the length of time that an unsuccessful
	// authorization response will be cached. You generally want more responsive,
	// "deny, try again" flows.
	WebhookCacheUnauthorizedTTL time.Duration
}

// NewStandaloneAuthorizationOptions returns a default set of standalone
// authorization options.
func NewStandaloneAuthorizationOptions() *StandaloneAuthorizationOptions {
	return &StandaloneAuthorizationOptions{AlwaysAllow: true}
}

// Validate checks that the configuration is valid.
func (s *StandaloneAuthorizationOptions) Validate() []error {
	return []error{}
}

// AddFlags add flags to configure the receiver's values to the provided
// FlagSet.
func (s *StandaloneAuthorizationOptions) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&s.AlwaysAllow, "authorization-always-allow", s.AlwaysAllow,
		"Whether to authorize all authenticated requests. This will take precedence over any other authorizer.")

	fs.BoolVar(&s.Webhook, "authorization-webhook", s.Webhook,
		"Whether to use a webhook to authorize requests.")
	fs.StringVar(&s.WebhookConfigFile, "authorization-webhook-config-file", s.WebhookConfigFile,
		"File with webhook configuration in kubeconfig format, used with --authorization-webhook=true. "+
			"The API server will query the remote service to determine access on the API server's secure port.")

	fs.DurationVar(&s.WebhookCacheAuthorizedTTL, "authorization-webhook-cache-authorized-ttl",
		s.WebhookCacheAuthorizedTTL,
		"The duration to cache 'authorized' responses from the webhook authorizer.")
	fs.DurationVar(&s.WebhookCacheUnauthorizedTTL,
		"authorization-webhook-cache-unauthorized-ttl", s.WebhookCacheUnauthorizedTTL,
		"The duration to cache 'unauthorized' responses from the webhook authorizer.")
}

// ApplyTo applies the configured options in the receiver to an apiserver
// Config.
func (s *StandaloneAuthorizationOptions) ApplyTo(c *genericapiserver.Config) error {
	authorizer, err := s.toAuthorizationConfig().New()
	if err != nil {
		return err
	}
	c.Authorizer = authorizer
	return nil
}

func (s *StandaloneAuthorizationOptions) toAuthorizationConfig() StandaloneAuthorizerConfig {
	return StandaloneAuthorizerConfig{
		AlwaysAllow:                 s.AlwaysAllow,
		Webhook:                     s.Webhook,
		WebhookConfigFile:           s.WebhookConfigFile,
		WebhookCacheAuthorizedTTL:   s.WebhookCacheAuthorizedTTL,
		WebhookCacheUnauthorizedTTL: s.WebhookCacheUnauthorizedTTL,
	}
}
