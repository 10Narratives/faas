package funcdomain

type Manifest struct {
	Name   string `yaml:"name"`
	Upload struct {
		SourceDir string `yaml:"source_dir"`
	} `yaml:"upload"`
}
