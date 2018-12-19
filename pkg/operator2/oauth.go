package operator2

import (
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kubecontrolplanev1 "github.com/openshift/api/kubecontrolplane/v1"
	osinv1 "github.com/openshift/api/osin/v1"
	"github.com/openshift/library-go/pkg/operator/resource/resourcemerge"
)

func (c *osinOperator) handleOAuthConfig(configOverrides []byte) (*corev1.ConfigMap, error) {
	oauthConfig, err := c.oauth.Get(configName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// TODO convert it to osin's cli config

	// TODO this pretends this is an OsinServerConfig
	cliConfig := &kubecontrolplanev1.KubeAPIServerConfig{
		OAuthConfig: &osinv1.OAuthConfig{
			MasterCA:                    nil,
			MasterURL:                   "",
			MasterPublicURL:             "",
			AssetPublicURL:              "",
			AlwaysShowProviderSelection: false,
			IdentityProviders:           nil,
			GrantConfig: osinv1.GrantConfig{
				Method:               "",
				ServiceAccountMethod: "",
			},
			SessionConfig: &osinv1.SessionConfig{
				SessionSecretsFile:   fmt.Sprintf("%s/%s", sessionPath, sessionKey),
				SessionMaxAgeSeconds: 5 * 60,
				SessionName:          "ssn",
			},
			TokenConfig: osinv1.TokenConfig{
				AuthorizeTokenMaxAgeSeconds:         0,
				AccessTokenMaxAgeSeconds:            0,
				AccessTokenInactivityTimeoutSeconds: nil,
			},
			Templates: &osinv1.OAuthTemplates{
				Login:             "",
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
