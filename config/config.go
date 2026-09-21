package config

import (
	"ibsTool/models"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type ConfigHandler struct {
	path string
}

func NewConfigHandler(path string) *ConfigHandler {
	return &ConfigHandler{path: path}
}

func (cH *ConfigHandler) GetAbsolutePath() string {
	p, _ := filepath.Abs(cH.path)
	return p
}

func (cH *ConfigHandler) ReadConfig() (settings *models.Settings, err error) {
	if _, err := os.Stat(cH.path); err != nil {
		return nil, err
	}

	b, err := os.ReadFile(cH.path)
	if err != nil {
		return nil, err
	}

	settings = &models.Settings{}

	if err := yaml.Unmarshal(b, settings); err != nil {
		return nil, err
	}

	return
}

func (cH *ConfigHandler) WriteConfig(settings models.Settings) error {

	b, err := yaml.Marshal(settings)
	if err != nil {
		return nil
	}

	if _, err := os.Stat(cH.path); err != nil {
		dir := filepath.Dir(cH.path)
		if err := os.MkdirAll(dir, 0664); err != nil {
			return err
		}
	}

	file, err := os.Create(cH.path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(b)
	if err != nil {
		return err
	}
	return nil
}
