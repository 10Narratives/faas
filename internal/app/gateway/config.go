package gatewayapp

type Config struct {
	Server         ServerConfig         `yaml:"server"`
	UnifiedStorage UnifiedStorageConfig `yaml:"unified_storage"`
}

type ServerConfig struct {
	Grpc    GrpcConfig    `yaml:"grpc"`
	Metrics MetricsConfig `yaml:"metrics"`
}

type GrpcConfig struct {
	Address string `yaml:"address"`
}

type MetricsConfig struct {
	Address string `yaml:"address"`
}

type UnifiedStorageConfig struct {
	URL string `yaml:"url" env-required:"true"`
}
