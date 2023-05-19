package config

import (
	"flag"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

var configPath = flag.String("c", "/usr/local/etc/conf.yaml", "config file path")

type ProxyConf struct {
	Server string `yaml:"server"`
	Probe  string `yaml:"probe"`
}
type HTTPServerConf struct {
	ListenAddr   string        `yaml:"listen_addr"`
	ReadTimeout  time.Duration `yaml:"http_read_timeout"`
	WriteTimeout time.Duration `yaml:"http_write_timeout"`
	MaxConn      int           `yaml:"max_conn"`
}

type Config struct {
	Debug bool            `yaml:"debug"`
	HTTP  *HTTPServerConf `yaml:"server"`
	Proxy []*ProxyConf    `yaml:"proxy"`
}

func GetConfig() (Config, error) {
	conf := Config{}
	flag.Parse()
	data, err := os.ReadFile(*configPath)
	if err != nil {
		return conf, err
	}
	if err := yaml.Unmarshal(data, &conf); err != nil {
		return conf, err
	}
	return conf, nil
}