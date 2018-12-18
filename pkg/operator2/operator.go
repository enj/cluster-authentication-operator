package operator2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	operatorv1 "github.com/openshift/api/operator/v1"
	"github.com/openshift/api/osin/v1"
	routeclient "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	osinv1alpha1 "github.com/openshift/cluster-osin-operator/pkg/apis/osin/v1alpha1"
	"github.com/openshift/cluster-osin-operator/pkg/boilerplate/operator"
	osinclient "github.com/openshift/cluster-osin-operator/pkg/generated/clientset/versioned/typed/osin/v1alpha1"
	osininformer "github.com/openshift/cluster-osin-operator/pkg/generated/informers/externalversions/osin/v1alpha1"
	"github.com/openshift/library-go/pkg/operator/events"
	"github.com/openshift/library-go/pkg/operator/resource/resourceapply"
)

const (
	targetName = "openshift-osin"
)

type osinOperator struct {
	osin osinclient.OsinInterface

	recorder events.Recorder

	route       routeclient.RouteInterface
	services    corev1.ServicesGetter
	secrets     corev1.SecretsGetter
	configMaps  corev1.ConfigMapsGetter
	deployments appsv1.DeploymentsGetter
}

func NewOsinOperator(informer osininformer.OsinInformer, getter osinclient.OsinsGetter, recorder events.Recorder) operator.Runner {
	c := &osinOperator{
		osin: getter.Osins(targetName),

		recorder: recorder,

		route:       nil, // TODO fix
		services:    nil,
		secrets:     nil,
		configMaps:  nil,
		deployments: nil,
	}

	return operator.New("OsinOperator2", c,
		operator.WithInformer(informer, operator.FilterByNames(targetName)),
	)
}

func (c *osinOperator) Key() (metav1.Object, error) {
	return c.osin.Get(targetName, metav1.GetOptions{})
}

func (c *osinOperator) Sync(obj metav1.Object) error {
	osinConfig := obj.(*osinv1alpha1.Osin)

	if osinConfig.Spec.ManagementState != operatorv1.Managed {
		return nil // TODO do something better for all states
	}

	if err := c.handleSync(osinConfig.Spec.UnsupportedConfigOverrides.Raw); err != nil {
		return err
	}

	// TODO update states and handle ClusterOperator spec/status

	return nil
}

func (c *osinOperator) handleSync(configOverrides []byte) error {
	route, err := c.handleRoute()
	if err != nil {
		return err
	}

	service, _, err := resourceapply.ApplyService(c.services, c.recorder, defaultService())
	if err != nil {
		return err
	}

	// session secret
	secret, _, err := resourceapply.ApplySecret(c.secrets, c.recorder, c.expectedSessionSecret())
	if err != nil {
		return err
	}

	// get / create default top level oauth config

	// convert it to osin's cli config

	// overlay configOverrides

	// write config map with full config bytes

	oauthConfig := &v1.OAuthConfig{}

	configMap, _, err := resourceapply.ApplyConfigMap(c.configMaps, c.recorder, nil)
	if err != nil {
		return err
	}

	// deployment, have RV of all resources
	// TODO use ExpectedDeploymentGeneration func
	// TODO probably do not need every RV
	expectedDeployment := defaultDeployment(route.ResourceVersion, service.ResourceVersion, secret.ResourceVersion, configMap.ResourceVersion)
	deployment, _, err := resourceapply.ApplyDeployment(c.deployments, c.recorder, expectedDeployment, c.getGeneration(), false)
	if err != nil {
		return err
	}

	return nil
}

func defaultLabels() map[string]string {
	return map[string]string{
		"app": "origin-cluster-osin-operator2",
	}
}

func defaultMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:            targetName,
		Namespace:       targetName,
		OwnerReferences: nil, // TODO
	}
}
