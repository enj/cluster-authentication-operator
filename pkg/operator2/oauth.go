package operator2

import (
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/golang/glog"

	configv1 "github.com/openshift/api/config/v1"
	kubecontrolplanev1 "github.com/openshift/api/kubecontrolplane/v1"
	osinv1 "github.com/openshift/api/osin/v1"
	"github.com/openshift/library-go/pkg/crypto"
	"github.com/openshift/library-go/pkg/operator/resource/resourcemerge"
)

func (c *osinOperator) handleOAuthConfig(configOverrides []byte) (*corev1.ConfigMap, []idpSyncData, error) {
	oauthConfig, err := c.oauth.Get(configName, metav1.GetOptions{})
	if err != nil {
		return nil, nil, err
	}

	// TODO remove this hack
	// pretend we have this one instead of the real input for now
	// this has the bits we need to care about
	oauthConfig.Spec.IdentityProviders = []configv1.IdentityProvider{
		{
			Name:            "happy",
			UseAsChallenger: true,
			UseAsLogin:      true,
			MappingMethod:   configv1.MappingMethodClaim,
			ProviderConfig: configv1.IdentityProviderConfig{
				Type: configv1.IdentityProviderTypeHTPasswd,
				HTPasswd: &configv1.HTPasswdIdentityProvider{
					FileData: configv1.SecretNameReference{
						Name: "fancy",
					},
				},
			},
		},
		{
			Name:            "new",
			UseAsChallenger: true,
			UseAsLogin:      true,
			MappingMethod:   configv1.MappingMethodClaim,
			ProviderConfig: configv1.IdentityProviderConfig{
				Type: configv1.IdentityProviderTypeOpenID,
				OpenID: &configv1.OpenIDIdentityProvider{
					ClientID: "panda",
					ClientSecret: configv1.SecretNameReference{
						Name: "fancy",
					},
					CA: configv1.ConfigMapNameReference{
						Name: "mah-ca",
					},
					URLs: configv1.OpenIDURLs{
						Authorize: "https://example.com/a",
						Token:     "https://example.com/b",
						UserInfo:  "https://example.com/c",
					},
					Claims: configv1.OpenIDClaims{
						PreferredUsername: []string{"preferred_username"},
						Name:              []string{"name"},
						Email:             []string{"email"},
					},
				},
			},
		},
	}

	// TODO maybe move the OAuth stuff up one level
	syncData, err := c.handleConfigSync(oauthConfig)
	if err != nil {
		return nil, nil, err
	}

	var accessTokenInactivityTimeoutSeconds *int32
	timeout := oauthConfig.Spec.TokenConfig.AccessTokenInactivityTimeoutSeconds
	switch {
	case timeout < 0:
		zero := int32(0)
		accessTokenInactivityTimeoutSeconds = &zero
	case timeout == 0:
		accessTokenInactivityTimeoutSeconds = nil
	case timeout > 0:
		accessTokenInactivityTimeoutSeconds = &timeout
	}

	var templates *osinv1.OAuthTemplates
	emptyTemplates := configv1.OAuthTemplates{}
	if oauthConfig.Spec.Templates != emptyTemplates {
		templates = &osinv1.OAuthTemplates{
			Login:             "", // TODO fix
			ProviderSelection: "", // TODO fix
			Error:             "", // TODO fix
		}
	}

	identityProviders := make([]osinv1.IdentityProvider, 0, len(oauthConfig.Spec.IdentityProviders))
	for i, idp := range oauthConfig.Spec.IdentityProviders {
		providerConfigBytes, err := convertProviderConfigToOsinBytes(&idp.ProviderConfig, syncData, i)
		if err != nil {
			glog.Error(err)
			continue
		}
		identityProviders = append(identityProviders,
			osinv1.IdentityProvider{
				Name:            idp.Name,
				UseAsChallenger: idp.UseAsChallenger,
				UseAsLogin:      idp.UseAsLogin,
				MappingMethod:   string(idp.MappingMethod),
				Provider: runtime.RawExtension{
					Raw:    providerConfigBytes,
					Object: nil, // grant config is incorrectly in the IDP, but should be dropped in general
				}, // TODO also need a series of config maps and secrets mounts based on this
			},
		)
	}
	if len(identityProviders) == 0 {
		identityProviders = []osinv1.IdentityProvider{
			createDenyAllIdentityProvider(),
		}
	}

	// TODO this pretends this is an OsinServerConfig
	cliConfig := &kubecontrolplanev1.KubeAPIServerConfig{
		GenericAPIServerConfig: configv1.GenericAPIServerConfig{
			ServingInfo: configv1.HTTPServingInfo{
				ServingInfo: configv1.ServingInfo{
					BindAddress: "0.0.0.0:443",
					BindNetwork: "tcp4",
					CertInfo: configv1.CertInfo{
						CertFile: "", // needs to be signed by MasterCA from below
						KeyFile:  "",
					},
					ClientCA:          "", // I think this can be left unset
					NamedCertificates: nil,
					MinTLSVersion:     crypto.TLSVersionToNameOrDie(crypto.DefaultTLSVersion()),
					CipherSuites:      crypto.CipherSuitesToNamesOrDie(crypto.DefaultCiphers()),
				},
				MaxRequestsInFlight:   1000,   // TODO this is a made up number
				RequestTimeoutSeconds: 5 * 60, // 5 minutes
			},
			CORSAllowedOrigins: nil,                    // TODO probably need this
			AuditConfig:        configv1.AuditConfig{}, // TODO probably need this
			KubeClientConfig: configv1.KubeClientConfig{
				KubeConfig: "", // this should use in cluster config
				ConnectionOverrides: configv1.ClientConnectionOverrides{
					QPS:   400, // TODO figure out values
					Burst: 400,
				},
			},
		},
		OAuthConfig: &osinv1.OAuthConfig{
			// TODO at the very least this needs to be set to self signed loopback CA for the token request endpoint
			MasterCA: nil,
			// TODO osin's code needs to be updated to properly use these values
			// it should use MasterURL in almost all places except the token request endpoint
			// which needs to direct the user to the real public URL (MasterPublicURL)
			// that means we still need to get that value from the installer's config
			// TODO ask installer team to make it easier to get that URL
			MasterURL:                   "https://127.0.0.1:443",
			MasterPublicURL:             "https://127.0.0.1:443",
			AssetPublicURL:              "", // TODO do we need this?
			AlwaysShowProviderSelection: false,
			IdentityProviders:           identityProviders,
			GrantConfig: osinv1.GrantConfig{
				Method:               osinv1.GrantHandlerPrompt, // TODO check
				ServiceAccountMethod: osinv1.GrantHandlerPrompt,
			},
			SessionConfig: &osinv1.SessionConfig{
				SessionSecretsFile:   fmt.Sprintf("%s/%s", sessionPath, sessionKey),
				SessionMaxAgeSeconds: 5 * 60, // 5 minutes
				SessionName:          "ssn",
			},
			TokenConfig: osinv1.TokenConfig{
				AuthorizeTokenMaxAgeSeconds:         5 * 60, // 5 minutes
				AccessTokenMaxAgeSeconds:            oauthConfig.Spec.TokenConfig.AccessTokenMaxAgeSeconds,
				AccessTokenInactivityTimeoutSeconds: accessTokenInactivityTimeoutSeconds,
			},
			Templates: templates,
		},
	}

	cliConfigBytes, err := json.Marshal(cliConfig)
	if err != nil {
		panic(err) // nothing in our config can fail to decode unless our scheme is broken, die
	}

	completeConfigBytes, err := resourcemerge.MergeProcessConfig(nil, cliConfigBytes, configOverrides)
	if err != nil {
		return nil, nil, err
	}

	return &corev1.ConfigMap{
		ObjectMeta: defaultMeta(),
		Data: map[string]string{
			configKey: string(completeConfigBytes),
		},
	}, syncData, nil
}
