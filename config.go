package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type config struct {
	AssetDir    string `json:"src"`
	DestName    string `json:"dest"`
	PackageName string `json:"package"`
	assetDirs   []string
}

func (conf *config) String() string {
	data, _ := json.MarshalIndent(conf, "", "    ")
	return string(data)
}

func parseConf() (*config, error) {
	confFilePath := flag.Arg(0)
	if confFilePath == "" {
		confFilePath = "asset.json"
	}
	_, err := os.Stat(confFilePath)
	var conf config
	if err == nil {
		data, err := ioutil.ReadFile(confFilePath)
		if err != nil {
			return nil, err
		}
		os.Chdir(filepath.Dir(confFilePath))
		err = json.Unmarshal(data, &conf)
		if err != nil {
			return nil, err
		}
	}
	if *src != "" {
		conf.AssetDir = *src
	}

	if *dest != "" {
		conf.DestName = *dest
	}
	if *packageName != "" {
		conf.PackageName = *packageName
	}
	if conf.AssetDir == "" {
		return nil, fmt.Errorf("asset src dir is empty")
	}

	if conf.DestName == "" {
		return nil, fmt.Errorf("asset dest is empty")
	}

	conf.assetDirs = strings.Split(conf.AssetDir, "|")
	for i, dir := range conf.assetDirs {
		if info, err := os.Stat(dir); err != nil {
			if !info.IsDir() {
				return nil, fmt.Errorf("asset dir[%s] is not dir", dir)
			}
			conf.assetDirs[i], _ = filepath.Abs(dir)
		}
	}

	destInfo, err := os.Stat(conf.DestName)

	if err == nil && destInfo.IsDir() {
		conf.DestName = conf.DestName + string(filepath.Separator) + "asset.go"
	}

	if conf.PackageName == "" {
		conf.PackageName = filepath.Base(conf.AssetDir)
	}

	return &conf, nil
}

var demoConf = `
{
  "src":"res",
  "dest":"res/assest.go",
  "package":"res"
}
`
