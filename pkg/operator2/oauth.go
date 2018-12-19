package operator2

import (
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	kubecontrolplanev1 "github.com/openshift/api/kubecontrolplane/v1"
	osinv1 "github.com/openshift/api/osin/v1"
	"github.com/openshift/library-go/pkg/operator/resource/resourcemerge"
)

func (c *osinOperator) handleOAuthConfig(configOverrides []byte) (*corev1.ConfigMap, error) {
	oauthConfig, err := c.oauth.Get(configName, metav1.GetOptions{})
	if err != nil {
		return nil, err
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

	identityProviders := make([]osinv1.IdentityProvider, 0, len(oauthConfig.Spec.IdentityProviders))
	for _, idp := range oauthConfig.Spec.IdentityProviders {
		identityProviders = append(identityProviders,
			osinv1.IdentityProvider{
				Name:            idp.Name,
				UseAsChallenger: idp.UseAsChallenger,
				UseAsLogin:      idp.UseAsLogin,
				MappingMethod:   string(idp.MappingMethod),
				Provider: runtime.RawExtension{
					Raw:    nil, // TODO write out all the tedious conversion logic
					Object: nil, // grant config is incorrectly in the IDP, but should be dropped in general
				}, // TODO also need a series of config maps and secrets mounts based on this
			},
		)
	}

	// TODO this pretends this is an OsinServerConfig
	cliConfig := &kubecontrolplanev1.KubeAPIServerConfig{
		OAuthConfig: &osinv1.OAuthConfig{
			MasterCA:                    nil, // TODO figure out if we need these to be set
			MasterURL:                   "",
			MasterPublicURL:             "",
			AssetPublicURL:              "",
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
				AuthorizeTokenMaxAgeSeconds:         oauthConfig.Spec.TokenConfig.AuthorizeTokenMaxAgeSeconds,
				AccessTokenMaxAgeSeconds:            oauthConfig.Spec.TokenConfig.AccessTokenMaxAgeSeconds,
				AccessTokenInactivityTimeoutSeconds: accessTokenInactivityTimeoutSeconds,
			},
			Templates: &osinv1.OAuthTemplates{
				Login:             "", // TODO need to handle these secrets here and in deployments
				ProviderSelection: "",
				Error:             "",
			},
		},
	}

	cliConfigBytes, err := json.Marshal(cliConfig)
	if err != nil {
		return nil, err
	}

	completeConfigBytes, err := resourcemerge.MergeProcessConfig(nil, cliConfigBytes, configOverrides)
	if err != nil {
		return nil, err
	}

	return &corev1.ConfigMap{
		ObjectMeta: defaultMeta(),
		Data: map[string]string{
			configKey: string(completeConfigBytes),
		},
	}, nil
}
