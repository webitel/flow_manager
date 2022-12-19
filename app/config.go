package app

import (
	"encoding/json"
	"io"
	"os"

	"github.com/BoRuDar/configuration/v4"
	"github.com/webitel/flow_manager/model"
)

func (f *FlowManager) Config() *model.Config {
	return f.config
}

func loadConfig() (*model.Config, error) {
	var config model.Config
	configurator := configuration.New(
		&config,
		configuration.NewEnvProvider(),
		configuration.NewFlagProvider(),
		configuration.NewDefaultProvider(),
	).SetOptions(configuration.OnFailFnOpt(func(err error) {
		//log.Println(err)
	}))

	if err := configurator.InitValues(); err != nil {
		//return nil, err
	}

	if config.ConfigFile != nil && *config.ConfigFile != "" {
		var body []byte
		f, err := os.OpenFile(*config.ConfigFile, os.O_RDONLY, 0644)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		if body, err = io.ReadAll(f); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(body, &config); err != nil {
			return nil, err
		}
	}

	return &config, nil
}
