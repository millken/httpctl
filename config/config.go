package config

import (
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/millken/httpctl/log"
	"github.com/pkg/errors"
)

var (
	// Default is the default config
	Default = Config{
		SubLogs: make(map[string]log.GlobalConfig),
	}
)

type (
	Http struct {
		Listen string `yaml:"listen" json:"listen"`
	}
	Https struct {
		Listen string `yaml:"listen" json:"listen"`
	}
	Server struct {
		Http     Http   `yaml:"http" json:"http"`
		Https    Https  `yaml:"https" json:"https"`
		Resolver string `yaml:"resolver" json:"resolver"`
	}
	ExampleExecutor struct {
		Enable bool `yaml:"enable" json:"enable"`
	}
	SiteCopyExecutor struct {
		Enable     bool     `yaml:"enable" json:"enable"`
		Hosts      []string `yaml:"hosts" json:"hosts"`
		OutputPath string   `yaml:"outputPath" json:"outputPath"`
	}
	SourceMapExecutor struct {
		Enable     bool     `yaml:"enable" json:"enable"`
		Hosts      []string `yaml:"hosts" json:"hosts"`
		OutputPath string   `yaml:"outputPath" json:"outputPath"`
	}
	FlowExecutor struct {
		Enable bool `yaml:"enable" json:"enable"`
	}
	Executor struct {
		Example   ExampleExecutor   `yaml:"example" json:"example"`
		SiteCopy  SiteCopyExecutor  `yaml:"sitecopy" json:"sitecopy"`
		SourceMap SourceMapExecutor `yaml:"sourcemap" json:"sourcemap"`
		Flow      FlowExecutor      `yaml:"flow" json:"flow"`
	}
	Config struct {
		Server   Server                      `yaml:"server" json:"server"`
		Log      log.GlobalConfig            `yaml:"log" json:"log"`
		SubLogs  map[string]log.GlobalConfig `yaml:"subLogs" json:"subLogs"`
		Executor Executor                    `yaml:"executor" json:"executor"`
	}
)

func New(path string) (cfg *Config, err error) {
	body, err := ioutil.ReadFile(path)
	if err != nil {
		return cfg, errors.Wrap(err, "failed to read config content")
	}
	fileExt := "yaml"
	extWithDot := filepath.Ext(path)
	if strings.HasPrefix(extWithDot, ".") {
		fileExt = extWithDot[1:]
	}
	cfg = &Default
	if err = Decode(body, cfg, fileExt); err != nil {
		return cfg, errors.Wrap(err, "failed to unmarshal config to struct")
	}
	return
}
