package config

import (
	"flag"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Server ServerConfig `yaml:"server"`
	Log LogConfig `yaml:"log"`
	Storage StorageConfig `yaml:"storage"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int `yaml:"port"`	
}

type LogConfig struct {
	Path string `yaml:"path"`
	Level int `yaml:"log_level"`
}

type StorageConfig struct {
	LinksSize int `yaml:"links_size"`
	CacheSize int `yaml:"cache_size"`
}

func MustLoad() *Config {
	path := loadPath()
	if path == "" {
		panic("Can`t read config file")
	}

	return loadConfig(path)
}

func loadPath() string {
	var path string
	flag.StringVar(&path, "config", "", "path to config file")
	flag.Parse()
	if path == "" {	
		path = "./config/config.yaml"
	}

	return path
}

func loadConfig(path string) *Config {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		panic("config file does not exist: " + path)
	}

	var cfg Config

	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		panic("cannot read config: " + err.Error())
	}
	
	return &cfg
}