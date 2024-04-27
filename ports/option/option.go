package option

type Option[Config any] interface {
	Configure(*Config)
}

type Func[Config any] func(*Config)

func (fn Func[Config]) Configure(c *Config) { fn(c) }

type initializable interface {
	Init()
}

func Use[Config any, OPT Option[Config]](opts []OPT) Config {
	var c Config
	if ic, ok := any(&c).(initializable); ok {
		ic.Init()
	}
	for _, opt := range opts {
		if any(opt) == nil {
			continue
		}
		opt.Configure(&c)
	}
	return c
}
