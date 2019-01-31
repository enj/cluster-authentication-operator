package controller

type InformerOption func() informerOption

type informerOption int

const (
	informerOptionSync informerOption = iota
	informerOptionNoSync
)

func WithSync() informerOption {
	return informerOptionSync
}

func WithoutSync() informerOption {
	return informerOptionNoSync
}

func toOption(opt informerOption, getter InformerGetter) Option {
	switch opt {
	case informerOptionSync:
		return WithInformerSynced(getter)
	case informerOptionNoSync:
		return func(*controller) {} // do nothing
	default:
		panic(opt)
	}
}
