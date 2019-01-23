package operator2

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	return ca, nil
}

func defaultCA() *corev1.ConfigMap {
	meta := defaultMeta()
	meta.Name = servingCAName
	meta.Annotations["service.alpha.openshift.io/inject-cabundle"] = "true"
	return &corev1.ConfigMap{
		ObjectMeta: meta,
	}
}
