package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"flag"
	"fmt"
	"go/format"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/css"
	"github.com/tdewolff/minify/js"
)

// VERSION current version
const VERSION = "0.6 20191001"

type staticFile struct {
	Name       string
	NameOrigin string
	Mtime      int64
	Content    string
}

var m *minify.M

func main() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "golang asset tool, version:", VERSION)
		fmt.Fprintln(os.Stderr, "https://github.com/hidu/goasset/")
		fmt.Fprintln(os.Stderr, "----------------------------------------------------------------------------------------")
		fmt.Fprintln(os.Stderr, "usage:")
		fmt.Fprintln(os.Stderr, "  goasset", " [-src=resource] [-dest=resource/asset.go] [-package=resource]  [asset.json]")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "config file (asset.json) example:")
		fmt.Fprintf(os.Stderr, demoConf)
	}

	flag.Parse()

	m = minify.New()
	m.AddFunc(".js", js.Minify)
	m.AddFunc(".css", css.Minify)

	conf, confErr := parseConf()
	if confErr != nil {
		log.Fatalln("parse config failed:", confErr)
	}

	err := conf.packAsset()
	if err != nil {
		log.Fatalln("pack asset with error: ", err)
	}
	log.Println("pack asset success")
}

var files []staticFile

func (conf *config) packAsset() error {
	log.Println("asset config:", conf)

	files = make([]staticFile, 0)

	for _, dir := range conf.assetDirs {
		filepath.Walk(dir, conf.walkerFor(dir))
	}

	var buf bytes.Buffer
	datas := make(map[string]interface{})
	datas["version"] = VERSION
	datas["files"] = files
	datas["package"] = conf.PackageName

	tpl.Execute(&buf, datas)
	codeBytes, err := format.Source(buf.Bytes())
	if err != nil {
		return err
	}

	//log.Println(strings.Repeat("-", 80))
	for _, staticFile := range files {
		log.Println("add", staticFile.NameOrigin)
	}
	//log.Println(strings.Repeat("-", 80))
	log.Println("total ", len(files), "assets")

	outFilePath := conf.DestName

	origin, err := ioutil.ReadFile(outFilePath)
	if err == nil && bytes.Equal(origin, codeBytes) {
		log.Println(outFilePath, "unchanged")
		return nil
	}
	err = ioutil.WriteFile(outFilePath, codeBytes, 0644)
	return err
}

func (conf *config) dataMinify(name string, data []byte) []byte {
	ext := filepath.Ext(name)
	if len(data) < 1 || ext == "" || (ext != ".js" && ext != ".css") || strings.HasSuffix(name, ".min"+ext) {
		return data
	}
	if bytes.Contains(data, []byte("no_minify")) {
		return data
	}
	d, err := m.Bytes(ext, data)
	if err != nil {
		log.Println("minify ", name, "failed, ignore it, err:", err)
		return data
	}

	log.Println("minify:", name, "(", len(data), "->", len(d), ")", fmt.Sprintf("  %.2f%%", float64(len(d))/float64(len(data))*100.0))
	return d
}

func (conf *config) walkerFor(baseDir string) filepath.WalkFunc {
	destName, _ := filepath.Abs(conf.DestName)
	return func(name string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			absName, errPath := filepath.Abs(name)
			if errPath != nil || absName == destName {
				return nil
			}

			nameRel, _ := filepath.Rel(baseDir, name)
			if isIgnoreFile(nameRel) {
				return nil
			}
			data, errFile := ioutil.ReadFile(name)
			if errFile != nil {
				return errFile
			}
			data = conf.dataMinify(name, data)
			nameSlash := string(filepath.Separator) + filepath.ToSlash(filepath.Base(baseDir)+string(filepath.Separator)+nameRel)
			nameSlash = strings.Replace(nameSlash, string(filepath.Separator), "/", -1)
			files = append(files, staticFile{
				Name:       base64.StdEncoding.EncodeToString([]byte(nameSlash)),
				NameOrigin: nameSlash,
				Content:    encode(data),
				Mtime:      info.ModTime().Unix(),
			})
		}

		return nil
	}
}

func encode(data []byte) string {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	gw.Write(data)
	gw.Flush()
	gw.Close()
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func isIgnoreFile(name string) bool {
	subNames := strings.Split(name, "/")
	for _, n := range subNames {
		if n[:1] == "." {
			return true
		}
	}
	return false
}
