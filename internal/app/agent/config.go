package agentapp

type Config struct {
	ObjectStorage ObjectStorageConfig `yaml:"object_storage"`
}

type ObjectStorageConfig struct {
	Endpoint            string `yaml:"endpoint" env-required:"true"`
	FunctionsBucketName string `yaml:"functions_bucket_name" env-default:"functions"`
	User                string `yaml:"user" env-required:"true"`
	Password            string `yaml:"password" env-required:"true"`
	UseSSL              bool   `yaml:"use_ssl" env-default:"false"`
}
