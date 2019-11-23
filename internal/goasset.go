package internal

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"go/format"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/css"
	"github.com/tdewolff/minify/js"
)

type staticFile struct {
	Name       string
	NameOrigin string
	Mtime      int64
	Content    []byte
	Size       int64
}

// HelperFunc 辅助方法的定义
type HelperFunc func(fileName string, content []byte) ([]byte, error)

// GoAsset go asset tool
type GoAsset struct {
	Config *Config
	Minify *minify.M
	Files  []staticFile
	Tpl    *template.Template
	Helper *assetHelper
}

func (ga *GoAsset) init() {
	ga.Minify = minify.New()
	ga.Minify.AddFunc(".js", js.Minify)
	ga.Minify.AddFunc(".css", css.Minify)

	ga.Files = make([]staticFile, 0)

	ga.Helper = newAssetHelper()

	tplTxt := Asset.GetContent("template/asset_tpl.go")
	ga.Tpl = template.Must(template.New("static").Funcs(template.FuncMap{
		"long_txt_fmt": func(txt []byte) string {
			encStr := base64.StdEncoding.EncodeToString(txt)
			reader := bytes.NewReader([]byte(encStr))
			chunk := make([]byte, 120)
			var buf bytes.Buffer
			for n, _ := reader.Read(chunk); n > 0; n, _ = reader.Read(chunk) {
				buf.WriteString("\n")
				buf.Write(chunk[0:n])
			}
			return buf.String()
		},
		"ts2str": func(ts int64) string {
			return time.Unix(ts, 0).Format("2006-01-02 15:04:05")
		},
	}).Parse(string(tplTxt)))
}

// Do main func
func (ga *GoAsset) Do() error {
	ga.init()

	log.Println("[goasset] asset Config:", ga.Config)

	wd, _ := os.Getwd()
	log.Println("[goasset] Current Dir:", wd)

	if err := ga.scan(); err != nil {
		return err
	}

	return ga.generate()
}

func (ga *GoAsset) scan() error {
	for _, dir := range ga.Config.assetDirs {
		if err := filepath.Walk(dir, ga.walkerFor(dir)); err != nil {
			return err
		}
	}

	// 按照文件名排序
	sort.Slice(ga.Files, func(i, j int) bool {
		a := ga.Files[i]
		b := ga.Files[j]
		return strings.Compare(a.Name, b.Name) < 0
	})

	for _, staticFile := range ga.Files {
		log.Println("[goasset] Add Asset:", staticFile.NameOrigin)
	}
	log.Println("[goasset] Total ", len(ga.Files), "Assets")
	return nil
}

func (ga *GoAsset) generate() error {
	var buf bytes.Buffer

	infos := make(map[string]interface{})
	infos["version"] = VERSION
	infos["files"] = ga.Files
	infos["package"] = ga.Config.PackageName
	infos["debug"] = ga.Config.Debug

	ga.Tpl.Execute(&buf, infos)

	codeBytes, err := format.Source(buf.Bytes())
	if err != nil {
		log.Println("[goasset] source code:\n", buf.String())
		return err
	}

	outFilePath := ga.Config.DestName

	origin, err := ioutil.ReadFile(outFilePath)
	if err == nil && bytes.Equal(origin, codeBytes) {
		log.Println("[goasset]", outFilePath, "unchanged")
		return nil
	}
	err = ioutil.WriteFile(outFilePath, codeBytes, 0644)
	return err
}

func (ga *GoAsset) walkerFor(baseDir string) filepath.WalkFunc {
	destName, _ := filepath.Abs(ga.Config.DestName)
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
			fileContent, errFile := ioutil.ReadFile(name)
			if errFile != nil {
				return errFile
			}
			contentNew, errHelper := ga.executeHelperFunc(absName, fileContent)
			if errHelper != nil {
				return errHelper
			}
			contentMin := ga.dataMinify(name, contentNew)
			nameSlash := string(filepath.Separator) + filepath.ToSlash(filepath.Base(baseDir)+string(filepath.Separator)+nameRel)
			nameSlash = strings.Replace(nameSlash, string(filepath.Separator), "/", -1)

			ga.Files = append(ga.Files, staticFile{
				Name:       nameSlash,
				NameOrigin: nameSlash,
				Content:    gzEncode(contentMin),
				Mtime:      info.ModTime().Unix(),
				Size:       info.Size(),
			})
		}
		return nil
	}
}

func (ga *GoAsset) executeHelperFunc(fileAbsPath string, content []byte) (contentNew []byte, err error) {
	return ga.Helper.Execute(fileAbsPath, content, "")
}

func (ga *GoAsset) dataMinify(name string, data []byte) []byte {
	ext := filepath.Ext(name)
	if len(data) < 1 || ext == "" || (ext != ".js" && ext != ".css") || strings.HasSuffix(name, ".min"+ext) {
		return data
	}
	if bytes.Contains(data, []byte("no_minify")) {
		return data
	}
	d, err := ga.Minify.Bytes(ext, data)
	if err != nil {
		log.Println("[goasset] minify ", name, "failed, ignore it, err:", err)
		return data
	}

	log.Println("[goasset] minify:", name, "(", len(data), "->", len(d), ")", fmt.Sprintf("  %.2f%%", float64(len(d))/float64(len(data))*100.0))
	return d
}

func gzEncode(data []byte) []byte {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	gw.Write(data)
	gw.Flush()
	gw.Close()
	return buf.Bytes()
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
