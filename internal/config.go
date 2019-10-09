package internal

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// Config 配置文件结构
type Config struct {
	AssetDir    string `json:"src"`
	DestName    string `json:"dest"`
	PackageName string `json:"package"`
	assetDirs   []string
}

func (conf *Config) String() string {
	data, _ := json.Marshal(conf)
	return string(data)
}

var resourceDir = flag.String("src", "resource/", "Asset Resource Dir, eg : resource/")
var destFileName = flag.String("dest", "resource/asset.go", "Destination FileName, eg : resource/asset.go ")
var packageName = flag.String("package", "resource", "Package Name, eg : resource")

// ParseConf 解析出配置
func ParseConf() (*Config, error) {
	confFilePath := flag.Arg(0)
	if confFilePath == "" {
		confFilePath = "asset.json"
	}
	_, err := os.Stat(confFilePath)
	conf := &Config{}
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
		if info, err := os.Stat(dir); err == nil {
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
