package operator2

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/library-go/pkg/operator/resourcesynccontroller"
)

func (c *osinOperator) handleConfigSync(_ *configv1.OAuth) ([]idpSyncData, error) {
	// pretend we have this one instead of the real input for now
	// this has the bits we need to care about
	config := &configv1.OAuth{
		Spec: configv1.OAuthSpec{
			IdentityProviders: []configv1.IdentityProvider{
				{
					Name: "happy",
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
					Name: "new",
					ProviderConfig: configv1.IdentityProviderConfig{
						Type: configv1.IdentityProviderTypeOpenID,
						OpenID: &configv1.OpenIDIdentityProvider{
							ClientSecret: configv1.SecretNameReference{
								Name: "fancy",
							},
							CA: configv1.ConfigMapNameReference{
								Name: "mah-ca",
							},
						},
					},
				},
			},
			Templates: configv1.OAuthTemplates{}, // later
		},
	}

	// TODO we probably need listers
	configMapClient := c.configMaps.ConfigMaps(userConfigNamespace)
	secretClient := c.secrets.Secrets(userConfigNamespace)

	configMaps, err := configMapClient.List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	secrets, err := secretClient.List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	prefixConfigMapNames := sets.NewString()
	prefixSecretNames := sets.NewString()

	// TODO this has too much boilerplate

	for _, cm := range configMaps.Items {
		if strings.HasPrefix(cm.Name, userConfigPrefix) {
			prefixConfigMapNames.Insert(cm.Name)
		}
	}

	for _, secret := range secrets.Items {
		if strings.HasPrefix(secret.Name, userConfigPrefix) {
			prefixSecretNames.Insert(secret.Name)
		}
	}

	inUseConfigMapNames := sets.NewString()
	inUseSecretNames := sets.NewString()

	data := convertToData(config.Spec.IdentityProviders)
	for _, d := range data {
		for dest, src := range d.configMaps {
			syncOrDie(c.resourceSyncer.SyncConfigMap, dest, src)
			inUseConfigMapNames.Insert(dest)
		}
		for dest, src := range d.secrets {
			syncOrDie(c.resourceSyncer.SyncSecret, dest, src)
			inUseSecretNames.Insert(dest)
		}
	}

	notInUseConfigMapNames := prefixConfigMapNames.Difference(inUseConfigMapNames)
	notInUseSecretNames := prefixSecretNames.Difference(inUseSecretNames)

	// TODO maybe update resource syncer in lib-go to cleanup its map as needed
	// it does not really matter, we are talking as worse case of a few unneeded strings
	for dest := range notInUseConfigMapNames {
		syncOrDie(c.resourceSyncer.SyncConfigMap, dest, "")
	}
	for dest := range notInUseSecretNames {
		syncOrDie(c.resourceSyncer.SyncSecret, dest, "")
	}

	return data, nil
}

type idpSyncData struct {
	// both maps are dest -> source
	// all strings are metadata.name
	configMaps map[string]string
	secrets    map[string]string
}

func convertToData(idps []configv1.IdentityProvider) []idpSyncData {
	out := make([]idpSyncData, 0, len(idps))
	for i, idp := range idps {
		pc := idp.ProviderConfig
		switch pc.Type {
		case configv1.IdentityProviderTypeHTPasswd:
			p := pc.HTPasswd // TODO could panic if invalid (applies to all)
			fileData := p.FileData.Name
			d := idpSyncData{
				secrets: map[string]string{
					getName(i, fileData, configv1.HTPasswdDataKey): fileData,
				},
			}
			out = append(out, d)
		case configv1.IdentityProviderTypeOpenID:
			p := pc.OpenID
			clientSecret := p.ClientSecret.Name
			ca := p.CA.Name
			d := idpSyncData{
				configMaps: map[string]string{
					getName(i, ca, corev1.ServiceAccountRootCAKey): ca,
				},
				secrets: map[string]string{
					getName(i, clientSecret, configv1.ClientSecretKey): clientSecret,
				},
			}
			out = append(out, d)
		default:
			panic("TODO")
		}
	}
	return out
}

const userConfigPrefix = "v4.0-config-user-idp-"

func getName(i int, name, key string) string {
	// TODO this scheme relies on each IDP struct not using the same key for more than one field
	// I think we can do better, but here is a start
	// A generic function that uses reflection may work too
	return fmt.Sprintf("%s%d-%s-%s", userConfigPrefix, i, name, key)
}

func syncOrDie(syncFunc func(dest, src resourcesynccontroller.ResourceLocation) error, dest, src string) {
	ns := userConfigNamespace
	if len(src) == 0 { // handle delete
		ns = ""
	}
	if err := syncFunc(
		resourcesynccontroller.ResourceLocation{
			Namespace: targetName, // TODO fix
			Name:      dest,
		},
		resourcesynccontroller.ResourceLocation{
			Namespace: ns,
			Name:      src,
		},
	); err != nil {
		panic(err) // implies incorrect informer wiring
	}
}
