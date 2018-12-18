package operator2

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/informers/internalinterfaces"
	"k8s.io/client-go/kubernetes"

	"github.com/openshift/cluster-osin-operator/pkg/apis/osin/v1alpha1"
	"github.com/openshift/cluster-osin-operator/pkg/generated/clientset/versioned"
	"github.com/openshift/cluster-osin-operator/pkg/generated/informers/externalversions"
	"github.com/openshift/library-go/pkg/controller/controllercmd"
	"github.com/openshift/library-go/pkg/operator/v1helpers"
)

const (
	resync = 20 * time.Minute

	osinResource = `
apiVersion: osin.openshift.io/v1alpha1
kind: Osin
metadata:
  name: openshift-osin
  namespace: openshift-osin
spec:
  managementState: Managed
`
)

func RunOperator(ctx *controllercmd.ControllerContext) error {
	kubeClient, err := kubernetes.NewForConfig(ctx.KubeConfig)
	if err != nil {
		return err
	}

	dynamicClient, err := dynamic.NewForConfig(ctx.KubeConfig)
	if err != nil {
		return err
	}

	osinClient, err := versioned.NewForConfig(ctx.KubeConfig)
	if err != nil {
		return err
	}

	kubeInformersNamespaced := informers.NewSharedInformerFactoryWithOptions(kubeClient, resync,
		informers.WithNamespace(targetName),
		informers.WithTweakListOptions(singleNameListOptions(targetName)),
	)

	osinInformersNamespaced := externalversions.NewSharedInformerFactoryWithOptions(osinClient, resync,
		externalversions.WithNamespace(targetName),
		externalversions.WithTweakListOptions(singleNameListOptions(targetName)),
	)

	v1helpers.EnsureOperatorConfigExists(
		dynamicClient,
		[]byte(osinResource),
		v1alpha1.GroupVersion.WithResource("osins"),
	)

	// TODO use kube informers/clients
	operator := NewOsinOperator(
		osinInformersNamespaced.Osin().V1alpha1().Osins(),
		osinClient.OsinV1alpha1(),
		recorder{}, // TODO ctx.EventRecorder,
	)

	kubeInformersNamespaced.Start(ctx.StopCh)
	osinInformersNamespaced.Start(ctx.StopCh)

	go operator.Run(ctx.StopCh)

	<-ctx.StopCh

	return fmt.Errorf("stopped")
}

func singleNameListOptions(name string) internalinterfaces.TweakListOptionsFunc {
	return func(opts *v1.ListOptions) {
		opts.FieldSelector = fields.OneTermEqualSelector("metadata.name", name).String()
	}
}

// temp hack until I fix lib-go
type recorder struct{}

func (recorder) Event(reason, message string)                            {}
func (recorder) Eventf(reason, messageFmt string, args ...interface{})   {}
func (recorder) Warning(reason, message string)                          {}
func (recorder) Warningf(reason, messageFmt string, args ...interface{}) {}
