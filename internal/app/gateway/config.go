package gatewayapp

type Config struct {
	Transport TransportConfig `yaml:"transport"`
	Databases DatabasesConfig `yaml:"databases"`
}

type TransportConfig struct {
	Grpc      GrpcConfig      `yaml:"grpc"`
	TaskQueue TaskQueueConfig `yaml:"task_queue"`
}

type GrpcConfig struct {
	Address string `yaml:"address"`
}

type TaskQueueConfig struct {
	URL string `yaml:"url" env-required:"true"`
}

type DatabasesConfig struct {
	StateDB       StateDBConfig       `yaml:"statedb"`
	ObjectStorage ObjectStorageConfig `yaml:"object_storage"`
}

type StateDBConfig struct {
	DSN string `yaml:"dsn" env-required:"true"`
}

type ObjectStorageConfig struct {
	Endpoint  string `yaml:"endpoint" env-required:"true"`
	AccessKey string `yaml:"access_key" env-required:"true"`
	SecretKey string `yaml:"secret_key" env-required:"true"`
}
