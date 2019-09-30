package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"flag"
	"fmt"
	"go/format"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/css"
	"github.com/tdewolff/minify/js"
)

// VERSION current version
const VERSION = "0.5.3 20161126"

type staticFile struct {
	Name       string
	NameOrigin string
	Mtime      int64
	Content    string
}

var src = flag.String("src", "", "asset src dir,eg : res/")
var dest = flag.String("dest", "", "dest file path,eg : res/asset.go ")
var packageName = flag.String("package", "", "package name,eg : res")

//var minifyFlag = flag.String("minify", "", "file need minify")

var m *minify.M

func main() {
	flag.Usage = func() {
		fmt.Println("usage:")
		fmt.Println("  goasset", " [-src=res] [-dest=demo] [-package=res]  [asset.json]")
		flag.PrintDefaults()
		fmt.Println("\ngolang asset tool,version:", VERSION)
		fmt.Println("https://github.com/hidu/goasset/\n")
		fmt.Println("json conf example:\n", demoConf)
	}
	m = minify.New()
	m.AddFunc(".js", js.Minify)
	m.AddFunc(".css", css.Minify)
	flag.Parse()
	conf, confErr := parseConf()
	if confErr != nil {
		fmt.Println("parse conf failed:", confErr, "\n")
		flag.Usage()
		os.Exit(1)
	}

	conf.packAsset()
}

var files []staticFile

func (conf *config) packAsset() {
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
		fmt.Println("go code err:\n", err, "\ncode is:\n")
		fmt.Println(buf.String())
		os.Exit(1)
	}
	fmt.Println("asset conf:")
	fmt.Println(conf.String())
	fmt.Println("total ", len(files), "assets")
	fmt.Println(strings.Repeat("-", 80))
	for _, staticFile := range files {
		fmt.Println("add", staticFile.NameOrigin)
	}
	fmt.Println(strings.Repeat("-", 80))

	outFilePath := conf.DestName

	origin, err := ioutil.ReadFile(outFilePath)
	if err == nil && bytes.Equal(origin, codeBytes) {
		fmt.Println(outFilePath, "unchanged")
		return
	}
	err = ioutil.WriteFile(outFilePath, codeBytes, 0644)
	if err == nil {
		fmt.Println("create ", outFilePath, "success")
	} else {
		fmt.Println("failed", err)
		os.Exit(2)
	}
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
		fmt.Println("minify ", name, "failed", err)
		return data
	}

	fmt.Println("minify:", name, len(data), "->", len(d), fmt.Sprintf("  %.2f%%", float64(len(d))/float64(len(data))*100.0))
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
