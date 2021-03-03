package core

import (
	"fmt"
	"github.com/pelletier/go-toml"
	"io"
	"io/ioutil"
)

type GitHub struct {
	AccessToken string `toml:"access-token"`
}

type Config struct {
	GitHub     GitHub
	SimnetDir  string `toml:"simnet-dir"`
	TestnetDir string `toml:"testnet-dir"`
	MainnetDir string `toml:"mainnet-dir"`
}

func parseConfig(reader io.Reader) (*Config, error) {
	config := Config{}
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	err = toml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	return &config, nil
}
