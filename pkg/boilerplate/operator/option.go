package operator

import (
	"reflect"

	"k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/cluster-authentication-operator/pkg/boilerplate/controller"
)

// key is the singleton key shared by all events
// the value is irrelevant, but pandas are awesome
const key = "🐼"

type Option func(*operator)

func WithInformer(getter controller.InformerGetter, filter controller.Filter, opts ...controller.InformerOption) Option {
	return toAppendOpt(
		controller.WithInformer(getter, controller.FilterFuncs{
			ParentFunc: func(obj v1.Object) (namespace, name string) {
				return key, key // return our singleton key for all events
			},
			AddFunc:    filter.Add,
			UpdateFunc: filter.Update,
			DeleteFunc: filter.Delete,
		}, opts...),
	)
}

func WithInitialEvent() Option {
	return toAppendOpt(
		controller.WithInitialEvent(key, key), // use singleton key for initial event
	)
}

func WithDefaultKey(key v1.Object) Option {
	return func(o *operator) {
		o.sync.key = reflect.ValueOf(key)
	}
}

func toAppendOpt(opt controller.Option) Option {
	return func(o *operator) {
		o.opts = append(o.opts, opt)
	}
}
