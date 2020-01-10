package template

// asset_remove_above()

// asset_include(header.tpl)

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// AssetFile one asset file
type AssetFile interface {
	Name() string
	ModTime() time.Time
	Content() []byte
	ContentGzip() []byte
}

// assetFile asset file  struct
type assetFile struct {
	name        string
	mtime       time.Time
	content     []byte
	contentGzip []byte
}

func (af *assetFile) Name() string {
	return af.name
}
func (af *assetFile) ModTime() time.Time {
	return af.mtime
}

func (af *assetFile) Content() []byte {
	return af.content
}
func (af *assetFile) ContentGzip() []byte {
	return af.contentGzip
}

var _ AssetFile = &assetFile{}

// AssetFiles asset files
type AssetFiles interface {
	GetAssetFile(name string) (AssetFile, error)
	GetContent(name string) []byte
	GetFileNames(dir string) []string

	FileHandlerFunc(name string) http.HandlerFunc
	HTTPHandler(baseDir string) http.Handler
}

// assetFiles asset files
type assetFiles struct {
	Files map[string]*assetFile
}

var _assetDirect bool

var _assetCwd, _ = os.Getwd()

// GetAssetFile get file by name
func (afs *assetFiles) GetAssetFile(name string) (AssetFile, error) {
	name = filepath.ToSlash(name)
	if name != "" && name[0] != '/' {
		name = "/" + name
	}
	if _assetDirect {
		assetFilePath := filepath.Join(_assetCwd, name)
		f, err := os.Open(assetFilePath)
		log.Println("[goasset] Asset Direct, name=", name, "assetPath=", assetFilePath, "err=", err)

		if err != nil {
			return nil, err
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil {
			return nil, err
		}
		if info.Mode().IsRegular() {
			content, err := ioutil.ReadAll(f)
			if err != nil {
				return nil, err
			}

			helper := newAssetHelper()
			contentNew, errHelper := helper.Execute(assetFilePath, content, "")
			if errHelper != nil {
				return nil, errHelper
			}
			return &assetFile{
				content: contentNew,
				name:    name,
				mtime:   info.ModTime(),
			}, nil
		}
		return nil, fmt.Errorf("not file")
	}
	if sf, has := afs.Files[name]; has {
		return sf, nil
	}
	return nil, fmt.Errorf("not exists")
}

// GetContent get content by name
func (afs *assetFiles) GetContent(name string) []byte {
	s, err := afs.GetAssetFile(name)
	if err != nil {
		return []byte("")
	}
	return s.Content()
}

// GetFileNames get all file names
func (afs *assetFiles) GetFileNames(dir string) []string {
	if dir == "" {
		dir = "/"
	}
	names := make([]string, 0, len(afs.Files))
	dirRaw := dir
	dir = path.Clean(dir)

	if dir != "/" && strings.HasSuffix(dirRaw, "/") {
		dir += string(filepath.Separator)
	}

	dir = filepath.ToSlash(dir)

	for name := range afs.Files {
		if strings.HasPrefix(name, dir) {
			names = append(names, name)
		}
	}
	return names
}

// FileHandlerFunc handler http files
// 若目录名称 为 *private 则不允许通过web访问
func (afs *assetFiles) FileHandlerFunc(name string) http.HandlerFunc {
	if strings.Contains(name, "private/") {
		return http.NotFound
	}
	return afs.FileHandlerFuncAll(name)
}

// FileHandlerFuncAll handler http files
// 无 private 目录规则
func (afs *assetFiles) FileHandlerFuncAll(name string) http.HandlerFunc {
	name = filepath.ToSlash(name)
	file, err := afs.GetAssetFile(name)
	return func(writer http.ResponseWriter, req *http.Request) {
		if err != nil {
			http.NotFound(writer, req)
			return
		}
		modifiedSince := req.Header.Get("If-Modified-Since")
		if modifiedSince != "" {
			t, err := time.Parse(http.TimeFormat, modifiedSince)
			if err == nil && file.ModTime().Before(t) {
				writer.Header().Del("Content-Type")
				writer.Header().Del("Content-Length")
				writer.Header().Set("Last-Modified", file.ModTime().UTC().Format(http.TimeFormat))
				writer.WriteHeader(http.StatusNotModified)
				return
			}
		}

		mimeType := mime.TypeByExtension(filepath.Ext(file.Name()))
		if mimeType != "" {
			writer.Header().Set("Content-Type", mimeType)
		}
		writer.Header().Set("Last-Modified", file.ModTime().UTC().Format(http.TimeFormat))

		gzipContent := file.ContentGzip()

		if len(gzipContent) > 0 && strings.Contains(req.Header.Get("Accept-Encoding"), "gzip") {
			writer.Header().Set("Content-Encoding", "gzip")
			writer.Write(gzipContent)
		} else {
			writer.Write(file.Content())
		}
	}
}

// HTTPHandler handler http request
// eg:on file system is :/res/js/a.js and request is /res/js/a.js
// http.Handle("/res/",res.Asset.HttpHandler("/"))

// eg:on file system is :/res/js/a.js and request is /js/a.js
// http.Handle("/js/",res.Asset.HttpHandler("/res/"))
func (afs *assetFiles) HTTPHandler(baseDir string) http.Handler {
	return &_assetFileServer{sf: afs, pdir: baseDir}
}

type _assetFileServer struct {
	sf   *assetFiles
	pdir string
}

// ServeHTTP ServeHTTP
func (f *_assetFileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name := filepath.ToSlash(filepath.Join(f.pdir, r.URL.Path))
	f.sf.FileHandlerFunc(name).ServeHTTP(w, r)
}

var _ AssetFiles = &assetFiles{}

var _ = flag.String
var _ = runtime.Version()

// ---------------------------helper.go--------begin--------------------------//
// asset_remove_start()
// regexp 包在当前文件并未使用，为了使当前模板import的包更整齐，故在此提前引入
func fixImportForHelper() {
	_ = regexp.Compile
	_ = gzip.ErrChecksum
	_ = base64.StdEncoding
	_ = bytes.TrimSpace
}

// asset_remove_end()

// asset_include(helper.go)
// ---------------------------helper.go--------finish-------------------------//

// Asset export assets
var Asset AssetFiles

// asset_include(footer.tpl)
