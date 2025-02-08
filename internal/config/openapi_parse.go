package config

// ParseConfig defines the parsing configuration for a service.
// MaxLevels is the maximum level to parse.
// MaxRecursionLevels is the maximum level to parse recursively.
// 0 means no recursion: property will get nil value.
// OnlyRequired is a flag whether to include only required fields.
// If the spec contains deep references, this might significantly speed up parsing.
type ParseConfig struct {
	MaxLevels          int  `koanf:"maxLevels" yaml:"maxLevels"`
	MaxRecursionLevels int  `koanf:"maxRecursionLevels" yaml:"maxRecursionLevels"`
	OnlyRequired       bool `koanf:"onlyRequired" yaml:"onlyRequired"`
}

func NewParseConfig() *ParseConfig {
	return &ParseConfig{
		MaxLevels: 0,
	}
}
