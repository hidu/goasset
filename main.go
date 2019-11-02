package main

//go:generate go run main.go -src template -package internal -dest internal/asset.go

//ignore_go:generate goasset -src template -package internal -dest internal/asset.go

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/hidu/goasset/internal"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "golang asset tool, version:", internal.VERSION)
		fmt.Fprintln(os.Stderr, "https://github.com/hidu/goasset/")
		fmt.Fprintln(os.Stderr, "----------------------------------------------------------------------------------------")
		fmt.Fprintln(os.Stderr, "usage:")
		fmt.Fprintln(os.Stderr, "  goasset", " [-src=resource] [-dest=resource/asset.go] [-package=resource]  [asset.json]")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "config file (asset.json) example:")
		fmt.Fprintln(os.Stderr, string(internal.Asset.GetContent("template/demo_config.json")))

	}

	flag.Parse()

	conf, confErr := internal.ParseConf()
	if confErr != nil {
		log.Fatalln("[goasset] parse config failed:", confErr)
	}
	ga := &internal.GoAsset{
		Config: conf,
	}

	err := ga.Do()

	if err != nil {
		log.Fatalln("[goasset] pack asset with error: ", err)
	}
	log.Println("[goasset] pack asset success")
}
