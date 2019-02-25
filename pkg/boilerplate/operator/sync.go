package operator

import (
	"reflect"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/cluster-authentication-operator/pkg/boilerplate/controller"
)

type KeySyncer interface {
	Key() (v1.Object, error)
	controller.Syncer
}

var _ controller.KeySyncer = &wrapper{}

type wrapper struct {
	KeySyncer
	key reflect.Value
}

func (s *wrapper) Key(_, _ string) (v1.Object, error) {
	obj, err := s.KeySyncer.Key()
	if errors.IsNotFound(err) && s.key.IsValid() {
		return reflect.New(s.key.Type()).Interface().(v1.Object), nil
	}
	return obj, err
}
