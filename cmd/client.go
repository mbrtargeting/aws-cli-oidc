package cmd

import (
	"net/url"

	"github.com/pkg/errors"

	"github.com/mbrtargeting/aws-cli-oidc/rest"
	input "github.com/natsukagami/go-input"
	"github.com/spf13/viper"
)

type RESTClient struct {
	client *rest.RestClient
}

type OIDCMetadataResponse struct {
	Issuer                                     string   `json:"issuer"`
	AuthorizationEndpoint                      string   `json:"authorization_endpoint"`
	TokenEndpoint                              string   `json:"token_endpoint"`
	TokenIntrospectionEndpoint                 string   `json:"token_introspection_endpoint"`
	UserinfoEndpoint                           string   `json:"userinfo_endpoint"`
	EndSessionEndpoint                         string   `json:"end_session_endpoint"`
	JwksURI                                    string   `json:"jwks_uri"`
	CheckSessionIframe                         string   `json:"check_session_iframe"`
	GrantTypesSupported                        []string `json:"grant_types_supported"`
	ResponseTypesSupported                     []string `json:"response_types_supported"`
	SubjectTypesSupported                      []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported           []string `json:"id_token_signing_alg_values_supported"`
	UserinfoSigningAlgValuesSupported          []string `json:"userinfo_signing_alg_values_supported"`
	RequestObjectSigningAlgValuesSupported     []string `json:"request_object_signing_alg_values_supported"`
	ResponseModesSupported                     []string `json:"response_modes_supported"`
	RegistrationEndpoint                       string   `json:"registration_endpoint"`
	TokenEndpointAuthMethodsSupported          []string `json:"token_endpoint_auth_methods_supported"`
	TokenEndpointAuthSigningAlgValuesSupported []string `json:"token_endpoint_auth_signing_alg_values_supported"`
	ClaimsSupported                            []string `json:"claims_supported"`
	ClaimTypesSupported                        []string `json:"claim_types_supported"`
	ClaimsParameterSupported                   bool     `json:"claims_parameter_supported"`
	ScopesSupported                            []string `json:"scopes_supported"`
	RequestParameterSupported                  bool     `json:"request_parameter_supported"`
	RequestURIParameterSupported               bool     `json:"request_uri_parameter_supported"`
	CodeChallengeMethodsSupported              []string `json:"code_challenge_methods_supported"`
	TLSClientCertificateBoundAccessTokens      bool     `json:"tls_client_certificate_bound_access_tokens"`
}

type OIDCClient struct {
	restClient *rest.RestClient
	base       *rest.WebTarget
	config     *viper.Viper
	metadata   *OIDCMetadataResponse
}

func InitializeClient(name string) (*OIDCClient, error) {
	config := viper.Sub(name)
	if config == nil {
		answer, _ := ui.Ask("OIDC provider URL not set, do you want to install a config file? [Y/n]", &input.Options{
			Default: "Y",
			Loop:    true,
			ValidateFunc: func(s string) error {
				if s != "Y" && s != "n" {
					return errors.New("Input must be Y or n")
				}
				return nil
			},
		})
		if answer == "n" {
			return nil, errors.New("Failed to initialize client because of no OIDC provider URL")
		}
		runSetup()
	}
	providerURL := config.GetString(OIDC_PROVIDER_METADATA_URL)

	restClient := rest.New()
	base := restClient.Target(providerURL)
	res, err := base.Request().Get()

	if err != nil {
		return nil, errors.Wrap(err, "Failed to get OIDC metadata")
	}

	if res.Status() != 200 {
		if res.MediaType() != "" {
			var json map[string]interface{}
			err := res.ReadJson(&json)
			if err == nil {
				return nil, errors.Errorf("Failed to get OIDC metadata, error: %s error_description: %s",
					json["error"], json["error_description"])
			}
		}
		return nil, errors.Errorf("Failed to get OIDC metadata, statusCode: %d", res.Status())
	}

	var metadata *OIDCMetadataResponse
	err = res.ReadJson(&metadata)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse OIDC metadata response")
	}

	client := &OIDCClient{restClient, base, config, metadata}

	if base == nil {
		return nil, errors.New("Failed to initialize client")
	}
	return client, nil
}

func (c *OIDCClient) ClientForm() url.Values {
	form := url.Values{}
	clientId := c.config.GetString(CLIENT_ID)
	form.Set("client_id", clientId)
	secret := c.config.GetString(CLIENT_SECRET)
	if secret != "" {
		form.Set("client_secret", secret)
	}
	return form
}

func (c *OIDCClient) Authorization() *rest.WebTarget {
	return c.restClient.Target(c.metadata.AuthorizationEndpoint)
}

func (c *OIDCClient) Token() *rest.WebTarget {
	return c.restClient.Target(c.metadata.TokenEndpoint)
}

func (c *OIDCClient) RedirectToSuccessfulPage() *rest.WebTarget {
	url := c.config.GetString(SUCCESSFUL_REDIRECT_URL)
	if url == "" {
		return nil
	}
	return c.restClient.Target(url)
}

func (c *OIDCClient) RedirectToFailurePage() *rest.WebTarget {
	url := c.config.GetString(FAILURE_REDIRECT_URL)
	if url == "" {
		return nil
	}
	return c.restClient.Target(url)
}
