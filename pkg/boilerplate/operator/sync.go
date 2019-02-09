package operator

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/cluster-authentication-operator/pkg/boilerplate/controller"
)

type KeySyncer interface {
	Key() (v1.Object, error)
	controller.Syncer
}

type DefaultKeyFunc func() v1.Object

var _ controller.KeySyncer = &wrapper{}

type wrapper struct {
	KeySyncer
	defaultKeyFunc DefaultKeyFunc
}

func (s *wrapper) Key(_, _ string) (v1.Object, error) {
	obj, err := s.KeySyncer.Key()
	if errors.IsNotFound(err) && s.defaultKeyFunc != nil {
		return s.defaultKeyFunc(), nil
	}
	return obj, err
}
