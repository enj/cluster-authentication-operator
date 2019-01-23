package operator2

import (
	"fmt"

	"github.com/golang/glog"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	injectCABundleAnnotationName  = "service.alpha.openshift.io/inject-cabundle"
	injectCABundleAnnotationValue = "true"
)

func (c *authOperator) handleCA() (*corev1.ConfigMap, error) {
	cm := c.configMaps.ConfigMaps(targetName)
	ca, err := cm.Get(servingCAName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		ca, err = cm.Create(defaultCA())
	}
	if err != nil {
		return nil, err
	}

	if len(ca.Data[servingCAKey]) == 0 {
		return nil, fmt.Errorf("config map has no ca data: %#v", ca)
	}

	if err := isValidCA(ca); err != nil {
		// delete the CA config map so that it is replaced with the proper one in next reconcile loop
		glog.Infof("deleting invalid ca config map %#v, deleteErr=%v", ca, cm.Delete(ca.Name, &metav1.DeleteOptions{}))
		return nil, err
	}

	return ca, nil
}

func isValidCA(ca *corev1.ConfigMap) error {
	if ca.Annotations[injectCABundleAnnotationName] != injectCABundleAnnotationValue {
		return fmt.Errorf("config map missing injection annotation: %#v", ca)
	}
	return nil
}

func defaultCA() *corev1.ConfigMap {
	meta := defaultMeta()
	meta.Name = servingCAName
	meta.Annotations[injectCABundleAnnotationName] = injectCABundleAnnotationValue
	return &corev1.ConfigMap{
		ObjectMeta: meta,
	}
}
