package config

import (
	"flag"
	"github.com/ilyakaznacheev/cleanenv"
	"os"
	"time"
)

type Config struct {
	GRPC GRPCConfig `yaml:"grpc"`
}

type GRPCConfig struct {
	Port    int `yaml:"port"`
	Timeout time.Duration
}

func MustLoad() *Config {
	path := fetchConfigPath()

	return MustLoadByPath(path)
}

func MustLoadByPath(path string) *Config {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		panic("Конфиг не найден! Путь: " + path)
	}

	var config Config
	if err := cleanenv.ReadConfig(path, &config); err != nil {
		panic(err)
	}
	return &config
}

func fetchConfigPath() string {
	var res string
	flag.StringVar(&res, "config", "", "path to config file")
	flag.Parse()

	if res == "" {
		res = ""
	}

	return res
}
