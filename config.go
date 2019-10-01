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
	data, _ := json.Marshal(conf)
	return string(data)
}

var resourceDir = flag.String("src", "resource/", "asset resource dir, eg : resource/")
var destFileName = flag.String("dest", "resource/asset.go", "dest FileName, eg : resource/asset.go ")
var packageName = flag.String("package", "resource", "package name, eg : resource")

func parseConf() (*config, error) {
	confFilePath := flag.Arg(0)
	if confFilePath == "" {
		confFilePath = "asset.json"
	}
	_, err := os.Stat(confFilePath)
	conf := &config{}
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
	} else {
		if *resourceDir != "" {
			conf.AssetDir = *resourceDir
		}

		if *destFileName != "" {
			conf.DestName = *destFileName
		}
		if *packageName != "" {
			conf.PackageName = *packageName
		}
	}
	if conf.AssetDir == "" {
		return nil, fmt.Errorf("asset resource dir is empty")
	}

	if conf.DestName == "" {
		return nil, fmt.Errorf("asset destFileName is empty")
	}

	conf.assetDirs = strings.Split(conf.AssetDir, "|")
	for idx, dir := range conf.assetDirs {
		if info, err := os.Stat(dir); err != nil {
			if !info.IsDir() {
				return nil, fmt.Errorf("asset dir[%s] is not dir", dir)
			}
			conf.assetDirs[idx], _ = filepath.Abs(dir)
		}
	}

	destInfo, err := os.Stat(conf.DestName)

	if err == nil && destInfo.IsDir() {
		conf.DestName = filepath.Join(conf.DestName, "asset.go")
	}

	if conf.PackageName == "" {
		conf.PackageName = filepath.Base(conf.AssetDir)
	}

	return conf, nil
}

var demoConf = `
{
  "src":"resource/",
  "dest":"resource/asset.go",
  "package":"resource"
}
`
