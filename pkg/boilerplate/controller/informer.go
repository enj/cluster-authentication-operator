package controller

type InformerOption func(InformerGetter, ParentFilter) Option

func WithSync() InformerOption {
	return func(getter InformerGetter, filter ParentFilter) Option {
		return WithInformerSynced(getter)
	}
}

func NoOption() InformerOption {
	return func(InformerGetter, ParentFilter) Option {
		return func(*controller) {} // do nothing
	}
}
