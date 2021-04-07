package utils

import (
	"github.com/mitchellh/go-homedir"
	"os"
	"path/filepath"
)

func GetHomeDirPath() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	return home, nil
}

func GetConfigPath() (string, error) {
	var err error
	cfgPath := os.Getenv(constants.BitchConfigPathEnvVarName)
	if len(cfgPath) == 0 {
		cfgPath, err = GetHomeDirPath()
		if err != nil {
			return "", err
		}
	}

	return filepath.Join(cfgPath, constants.BitchConfigDirName), nil
}

func CreateConfigPath() error {
	var err error
	cfgPath, err := GetConfigPath()
	if err != nil {
		return err
	}
	err = CreateDirRecursive(cfgPath)
	if err != nil {
		return err
	}
	for _, dir := range []string{constants.BitchImportCacheDir} {
		err := CreateDirRecursive(filepath.Join(cfgPath, dir))
		if err != nil {
			return err
		}
	}
	return nil
}
