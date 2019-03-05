package operator2

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	configv1 "github.com/openshift/api/config/v1"
	osinv1 "github.com/openshift/api/osin/v1"
)

const (
	caField = "ca"

	tlsClientCertField = "tls-client-cert"
	tlsClientKeyField  = "tls-client-key"

	clientSecretField = "client-secret"
)

var (
	scheme  = runtime.NewScheme()
	codecs  = serializer.NewCodecFactory(scheme)
	encoder = codecs.LegacyCodec(osinv1.GroupVersion) // TODO I think there is a better way to do this
)

func init() {
	utilruntime.Must(osinv1.Install(scheme))
}

func convertProviderConfigToOsinBytes(providerConfig *configv1.IdentityProviderConfig, syncData *configSyncData, i int) ([]byte, error) {
	const missingProviderFmt string = "type %s was specified, but its configuration is missing"

	var p runtime.Object

	switch providerConfig.Type {
	case configv1.IdentityProviderTypeBasicAuth:
		basicAuthConfig := providerConfig.BasicAuth
		if basicAuthConfig == nil {
			return nil, fmt.Errorf(missingProviderFmt, providerConfig.Type)
		}

		p = &osinv1.BasicAuthPasswordIdentityProvider{
			RemoteConnectionInfo: configv1.RemoteConnectionInfo{
				URL: basicAuthConfig.URL,
				CA:  syncData.addIDPConfigMap(i, basicAuthConfig.CA, caField, corev1.ServiceAccountRootCAKey),
				CertInfo: configv1.CertInfo{
					CertFile: syncData.addIDPSecret(i, basicAuthConfig.TLSClientCert, tlsClientCertField, corev1.TLSCertKey),
					KeyFile:  syncData.addIDPSecret(i, basicAuthConfig.TLSClientKey, tlsClientKeyField, corev1.TLSPrivateKeyKey),
				},
			},
		}

	case configv1.IdentityProviderTypeGitHub:
		githubConfig := providerConfig.GitHub
		if githubConfig == nil {
			return nil, fmt.Errorf(missingProviderFmt, providerConfig.Type)
		}

		p = &osinv1.GitHubIdentityProvider{
			ClientID:      githubConfig.ClientID,
			ClientSecret:  createFileStringSource(syncData.addIDPSecret(i, githubConfig.ClientSecret, clientSecretField, configv1.ClientSecretKey)),
			Organizations: githubConfig.Organizations,
			Teams:         githubConfig.Teams,
			Hostname:      githubConfig.Hostname,
			CA:            syncData.addIDPConfigMap(i, githubConfig.CA, caField, corev1.ServiceAccountRootCAKey),
		}

	case configv1.IdentityProviderTypeGitLab:
		gitlabConfig := providerConfig.GitLab
		if gitlabConfig == nil {
			return nil, fmt.Errorf(missingProviderFmt, providerConfig.Type)
		}

		p = &osinv1.GitLabIdentityProvider{
			CA:           syncData.addIDPConfigMap(i, gitlabConfig.CA, caField, corev1.ServiceAccountRootCAKey),
			URL:          gitlabConfig.URL,
			ClientID:     gitlabConfig.ClientID,
			ClientSecret: createFileStringSource(syncData.addIDPSecret(i, gitlabConfig.ClientSecret, clientSecretField, configv1.ClientSecretKey)),
			Legacy:       new(bool), // we require OIDC for GitLab now
		}

	case configv1.IdentityProviderTypeGoogle:
		googleConfig := providerConfig.Google
		if googleConfig == nil {
			return nil, fmt.Errorf(missingProviderFmt, providerConfig.Type)
		}

		p = &osinv1.GoogleIdentityProvider{
			ClientID:     googleConfig.ClientID,
			ClientSecret: createFileStringSource(syncData.addIDPSecret(i, googleConfig.ClientSecret, clientSecretField, configv1.ClientSecretKey)),
			HostedDomain: googleConfig.HostedDomain,
		}

	case configv1.IdentityProviderTypeHTPasswd:
		if providerConfig.HTPasswd == nil {
			return nil, fmt.Errorf(missingProviderFmt, providerConfig.Type)
		}

		p = &osinv1.HTPasswdPasswordIdentityProvider{
			File: syncData.addIDPSecret(i, providerConfig.HTPasswd.FileData, "file-data", configv1.HTPasswdDataKey),
		}

	case configv1.IdentityProviderTypeKeystone:
		keystoneConfig := providerConfig.Keystone
		if keystoneConfig == nil {
			return nil, fmt.Errorf(missingProviderFmt, providerConfig.Type)
		}

		p = &osinv1.KeystonePasswordIdentityProvider{
			RemoteConnectionInfo: configv1.RemoteConnectionInfo{
				URL: keystoneConfig.URL,
				CA:  syncData.addIDPConfigMap(i, keystoneConfig.CA, caField, corev1.ServiceAccountRootCAKey),
				CertInfo: configv1.CertInfo{
					CertFile: syncData.addIDPSecret(i, keystoneConfig.TLSClientCert, tlsClientCertField, corev1.TLSCertKey),
					KeyFile:  syncData.addIDPSecret(i, keystoneConfig.TLSClientKey, tlsClientKeyField, corev1.TLSPrivateKeyKey),
				},
			},
			DomainName:          keystoneConfig.DomainName,
			UseKeystoneIdentity: true, // force use of keystone ID
		}

	case configv1.IdentityProviderTypeLDAP:
		ldapConfig := providerConfig.LDAP
		if ldapConfig == nil {
			return nil, fmt.Errorf(missingProviderFmt, providerConfig.Type)
		}

		p = &osinv1.LDAPPasswordIdentityProvider{
			URL:          ldapConfig.URL,
			BindDN:       ldapConfig.BindDN,
			BindPassword: createFileStringSource(syncData.addIDPSecret(i, ldapConfig.BindPassword, "bind-password", configv1.BindPasswordKey)),
			Insecure:     ldapConfig.Insecure,
			CA:           syncData.addIDPConfigMap(i, ldapConfig.CA, caField, corev1.ServiceAccountRootCAKey),
			Attributes: osinv1.LDAPAttributeMapping{
				ID:                ldapConfig.Attributes.ID,
				PreferredUsername: ldapConfig.Attributes.PreferredUsername,
				Name:              ldapConfig.Attributes.Name,
				Email:             ldapConfig.Attributes.Email,
			},
		}

	case configv1.IdentityProviderTypeOpenID:
		openIDConfig := providerConfig.OpenID
		if openIDConfig == nil {
			return nil, fmt.Errorf(missingProviderFmt, providerConfig.Type)
		}

		p = &osinv1.OpenIDIdentityProvider{
			CA:                       syncData.addIDPConfigMap(i, openIDConfig.CA, caField, corev1.ServiceAccountRootCAKey),
			ClientID:                 openIDConfig.ClientID,
			ClientSecret:             createFileStringSource(syncData.addIDPSecret(i, openIDConfig.ClientSecret, clientSecretField, configv1.ClientSecretKey)),
			ExtraScopes:              openIDConfig.ExtraScopes,
			ExtraAuthorizeParameters: openIDConfig.ExtraAuthorizeParameters,
			URLs: osinv1.OpenIDURLs{
				Authorize: openIDConfig.URLs.Authorize,
				Token:     openIDConfig.URLs.Token,
				UserInfo:  openIDConfig.URLs.UserInfo,
			},
			Claims: osinv1.OpenIDClaims{
				// There is no longer a user-facing setting for ID as it is considered unsafe
				ID:                []string{configv1.UserIDClaim},
				PreferredUsername: openIDConfig.Claims.PreferredUsername,
				Name:              openIDConfig.Claims.Name,
				Email:             openIDConfig.Claims.Email,
			},
		}

	case configv1.IdentityProviderTypeRequestHeader:
		requestHeaderConfig := providerConfig.RequestHeader
		if requestHeaderConfig == nil {
			return nil, fmt.Errorf(missingProviderFmt, providerConfig.Type)
		}

		p = &osinv1.RequestHeaderIdentityProvider{
			LoginURL:                 requestHeaderConfig.LoginURL,
			ChallengeURL:             requestHeaderConfig.ChallengeURL,
			ClientCA:                 syncData.addIDPConfigMap(i, requestHeaderConfig.ClientCA, caField, corev1.ServiceAccountRootCAKey),
			ClientCommonNames:        requestHeaderConfig.ClientCommonNames,
			Headers:                  requestHeaderConfig.Headers,
			PreferredUsernameHeaders: requestHeaderConfig.PreferredUsernameHeaders,
			NameHeaders:              requestHeaderConfig.NameHeaders,
			EmailHeaders:             requestHeaderConfig.EmailHeaders,
		}

	default:
		return nil, fmt.Errorf("the identity provider type '%s' is not supported", providerConfig.Type)
	} // switch

	return encodeOrDie(p), nil
}

func createFileStringSource(filepath string) configv1.StringSource {
	return configv1.StringSource{
		StringSourceSpec: configv1.StringSourceSpec{
			File: filepath,
		},
	}
}

func encodeOrDie(obj runtime.Object) []byte {
	bytes, err := runtime.Encode(encoder, obj)
	if err != nil {
		panic(err) // indicates static generated code is broken, unrecoverable
	}
	return bytes
}
