package manifest

type Package struct {
	ID       string `yaml:"id" json:"id"`
	Category string `yaml:"category,omitempty" json:"category,omitempty"`
	Optional bool   `yaml:"optional,omitempty" json:"optional,omitempty"`
}

type Settings struct {
	Proxy        string `yaml:"proxy,omitempty" json:"proxy,omitempty"`
	SkipExisting bool   `yaml:"skip_existing,omitempty" json:"skip_existing,omitempty"`
	RetryCount   int    `yaml:"retry_count,omitempty" json:"retry_count,omitempty"`
	RetryDelay   int    `yaml:"retry_delay,omitempty" json:"retry_delay,omitempty"`
}

type Manifest struct {
	Settings Settings  `yaml:"settings,omitempty" json:"settings,omitempty"`
	Packages []Package `yaml:"packages" json:"packages"`
}
