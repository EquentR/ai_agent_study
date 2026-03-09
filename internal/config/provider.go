package config

type Provider interface {
	ModelName() string
	BaseURL() string
	Type() string
	AuthKey() string
}

type BaseProvider struct {
	Model   string `yaml:"model"`
	BaseUrl string `yaml:"baseUrl"`
	Typ     string `yaml:"type"`
	Key     string `yaml:"authKey"`
}

func (p *BaseProvider) ModelName() string {
	return p.Model
}

func (p *BaseProvider) BaseURL() string {
	return p.BaseUrl
}

func (p *BaseProvider) Type() string {
	return p.Typ
}

func (p *BaseProvider) AuthKey() string {
	return p.Key
}

type LLMProvider struct {
	BaseProvider `yaml:",inline"`
}

type EmbeddingProvider struct {
	BaseProvider `yaml:",inline"`
	Dimension    int `yaml:"dimension"`
}

type RerankingProvider struct {
	BaseProvider `yaml:",inline"`
}
