package operator

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/informers/internalinterfaces"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/openshift/cluster-authentication-operator/pkg/boilerplate/controller"
	"github.com/openshift/library-go/pkg/controller/controllercmd"
)

const resync = 20 * time.Minute

var oldKubeAPIServerOperatorConfigGVR = schema.GroupVersionResource{
	Group:    "kubeapiserver.operator.openshift.io",
	Version:  "v1alpha1",
	Resource: "kubeapiserveroperatorconfigs",
}
var kubeAPIServerOperatorConfigGVR = schema.GroupVersionResource{
	Group:    "operator.openshift.io",
	Version:  "v1",
	Resource: "kubeapiservers",
}

func RunOperator(ctx *controllercmd.ControllerContext) error {
	kubeClient, err := kubernetes.NewForConfig(ctx.KubeConfig)
	if err != nil {
		return err
	}

	dynamicClient, err := dynamic.NewForConfig(ctx.KubeConfig)
	if err != nil {
		return err
	}

	kubeInformersNamespaced := informers.NewSharedInformerFactoryWithOptions(kubeClient, resync,
		informers.WithNamespace(targetNamespaceName),
		informers.WithTweakListOptions(singleNameListOptions(targetConfigMap)),
	)

	oldKubeAPIServerOperatorConfig := dynamicClient.Resource(oldKubeAPIServerOperatorConfigGVR)
	oldKubeAPIServerOperatorConfigInformer := dynamicInformer(oldKubeAPIServerOperatorConfig)
	kubeAPIServerOperatorConfig := dynamicClient.Resource(kubeAPIServerOperatorConfigGVR)
	kubeAPIServerOperatorConfigInformer := dynamicInformer(kubeAPIServerOperatorConfig)

	operator := NewOsinOperator(
		kubeInformersNamespaced.Core().V1().ConfigMaps(),
		kubeClient.CoreV1(),
		oldKubeAPIServerOperatorConfigInformer,
		oldKubeAPIServerOperatorConfig,
		kubeAPIServerOperatorConfigInformer,
		kubeAPIServerOperatorConfig,
	)

	kubeInformersNamespaced.Start(ctx.StopCh)
	go oldKubeAPIServerOperatorConfigInformer.Informer().Run(ctx.StopCh)
	go kubeAPIServerOperatorConfigInformer.Informer().Run(ctx.StopCh)

	go operator.Run(ctx.StopCh)

	<-ctx.StopCh

	return fmt.Errorf("stopped")
}

func singleNameListOptions(name string) internalinterfaces.TweakListOptionsFunc {
	return func(opts *v1.ListOptions) {
		opts.FieldSelector = fields.OneTermEqualSelector("metadata.name", name).String()
	}
}

func dynamicInformer(resource dynamic.ResourceInterface) controller.InformerGetter {
	lw := &cache.ListWatch{
		ListFunc: func(opts v1.ListOptions) (runtime.Object, error) {
			return resource.List(opts)
		},
		WatchFunc: func(opts v1.ListOptions) (watch.Interface, error) {
			return resource.Watch(opts)
		},
	}
	informer := cache.NewSharedIndexInformer(lw, &unstructured.Unstructured{}, resync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	return &toInformerGetter{informer: informer}
}

type toInformerGetter struct {
	informer cache.SharedIndexInformer
}

func (g *toInformerGetter) Informer() cache.SharedIndexInformer {
	return g.informer
}
