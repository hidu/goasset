package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/css"
	"github.com/tdewolff/minify/js"
	"go/format"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// VERSION current version
const VERSION = "0.5.3 20161126"

type staticFile struct {
	Name       string
	NameOrigin string
	Mtime      int64
	Content    string
}

type assestConf struct {
	AssestDir   string `json:"src"`
	DestName    string `json:"dest"`
	PackageName string `json:"package"`
	//	Minify []string `json:"minify"`
	assestDirs []string
}

func (conf *assestConf) String() string {
	data, _ := json.MarshalIndent(conf, "", "    ")
	return string(data)
}

var src = flag.String("src", "", "assest src dir,eg : res/")
var dest = flag.String("dest", "", "dest file path,eg : res/assest.go ")
var packageName = flag.String("package", "", "package name,eg : res")

//var minifyFlag = flag.String("minify", "", "file need minify")

func parseConf() (*assestConf, error) {
	confFilePath := flag.Arg(0)
	if confFilePath == "" {
		confFilePath = "assest.json"
	}
	_, err := os.Stat(confFilePath)
	var conf assestConf
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
		conf.AssestDir = *src
	}

	if *dest != "" {
		conf.DestName = *dest
	}
	if *packageName != "" {
		conf.PackageName = *packageName
	}
	if conf.AssestDir == "" {
		return nil, fmt.Errorf("assest src dir is empty")
	}

	if conf.DestName == "" {
		return nil, fmt.Errorf("assest dest is empty")
	}

	conf.assestDirs = strings.Split(conf.AssestDir, "|")
	for i, dir := range conf.assestDirs {
		if info, err := os.Stat(dir); err != nil {
			if !info.IsDir() {
				return nil, fmt.Errorf("assest dir[%s] is not dir", dir)
			}
			conf.assestDirs[i], _ = filepath.Abs(dir)
		}
	}

	destInfo, err := os.Stat(conf.DestName)

	if err == nil && destInfo.IsDir() {
		conf.DestName = conf.DestName + string(filepath.Separator) + "assest.go"
	}

	if conf.PackageName == "" {
		conf.PackageName = filepath.Base(conf.AssestDir)
	}

	return &conf, nil
}

var m *minify.M

func main() {
	flag.Usage = func() {
		fmt.Println("useage:")
		fmt.Println("  goassest", " [-src=res] [-dest=demo] [-package=res]  [assest.json]")
		flag.PrintDefaults()
		fmt.Println("\ngolang assest tool,version:", VERSION)
		fmt.Println("https://github.com/hidu/goassest/\n")
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

	conf.makeAssest()
}

var files []staticFile

func (conf *assestConf) makeAssest() {
	files = make([]staticFile, 0)

	for _, dir := range conf.assestDirs {
		filepath.Walk(dir, conf.walkerFor(dir))
	}

	var buf bytes.Buffer
	datas := make(map[string]interface{})
	datas["version"] = VERSION
	datas["files"] = files
	datas["package"] = conf.PackageName
	//	datas["assestDir"] = conf.AssestDir

	tpl.Execute(&buf, datas)
	codeBytes, err := format.Source(buf.Bytes())
	if err != nil {
		fmt.Println("go code err:\n", err, "\ncode is:\n")
		fmt.Println(buf.String())
		os.Exit(1)
	}
	fmt.Println("assest conf:")
	fmt.Println(conf.String())
	fmt.Println("total ", len(files), "assests")
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

func (conf *assestConf) dataMinify(name string, data []byte) []byte {
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

func (conf *assestConf) walkerFor(baseDir string) filepath.WalkFunc {
	destName, _ := filepath.Abs(conf.DestName)
	return func(name string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			absName, err := filepath.Abs(name)
			if err != nil || absName == destName {
				return nil
			}

			nameRel, _ := filepath.Rel(baseDir, name)
			if isIgnoreFile(nameRel) {
				return nil
			}
			data, ferr := ioutil.ReadFile(name)
			if ferr != nil {
				return ferr
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

var tpl = template.Must(template.New("static").Parse(`
// generated by goassest({{$.version}})
// https://github.com/hidu/goassest/

package {{.package}}

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
	"runtime"
)

// AssestFile assest file  struct
type AssestFile struct{
	Name string
	Mtime int64
	Content string
}

// AssestStruct assest files
type AssestStruct struct{
  Files  map[string]*AssestFile
}

var _assestDirect bool

func init(){
	exeName:=filepath.Base(os.Getenv("_"))
	//only enable with go run
	if(exeName=="go" || (runtime.GOOS=="windows" && strings.Contains(os.Args[0], "go-build") ) ){
		flag.BoolVar(&_assestDirect, "assest_direct", false, "for debug,read assest direct")
	}
}

var _assestCwd,_=os.Getwd()

// GetAssestFile get file by name
func (statics *AssestStruct)GetAssestFile(name string) (*AssestFile,error){
	name=filepath.ToSlash(name)
	if(name!="" && name[0]!='/'){
		name="/"+name
	}
	if _assestDirect {
		f,err:=os.Open(filepath.Join(_assestCwd,name))
		if(err!=nil){
			return nil,err
		}
		defer f.Close()
		info,err:=f.Stat()
		if(err!=nil){
			return nil,err
		}
		if(info.Mode().IsRegular()){
			content,err:=ioutil.ReadAll(f)
			if(err!=nil){
				return nil,err
			}
			return &AssestFile{
				Content:string(content),
				Name:name,
				Mtime:info.ModTime().Unix(),
			},nil
		}
		return nil,fmt.Errorf("not file")
	}
	if sf,has:=statics.Files[name];has{
		return sf,nil
	}
	return nil,fmt.Errorf("not exists")
}

// GetContent get content by name
func (statics AssestStruct)GetContent(name string)string{
	s,err:=statics.GetAssestFile(name)
	if(err!=nil){
		return ""
	}
	return s.Content
}

// GetFileNames get all file names
func (statics AssestStruct)GetFileNames(dir string)[]string{
	if(dir==""){
		dir="/"
	}
	names:=make([]string,len(statics.Files))
		dirRaw:=dir
	dir = path.Clean(dir) 
	
	if(dir!="/" && strings.HasSuffix(dirRaw,"/")){
		dir+=string(filepath.Separator)
	}
	 
	dir=filepath.ToSlash(dir)
	
	for name:=range statics.Files{
		if(strings.HasPrefix(name,dir)){
			names=append(names,name)
		}
	}
	return names
}

// FileHandlerFunc handler http files
func (statics *AssestStruct)FileHandlerFunc(name string) http.HandlerFunc{
	if(strings.Contains(name,"private")){
		return http.NotFound
	}
	name=filepath.ToSlash(name)
	static, err := statics.GetAssestFile(name)
	return func(w http.ResponseWriter,r *http.Request){
		if(err!=nil){
			http.NotFound(w, r)
			return
		}
		modtime := time.Unix(static.Mtime, 0)
		modifiedSince := r.Header.Get("If-Modified-Since")
		if modifiedSince != "" {
			t, err := time.Parse(http.TimeFormat, modifiedSince)
			if err == nil && modtime.Before(t.Add(1*time.Second)) {
				w.Header().Del("Content-Type")
				w.Header().Del("Content-Length")
				w.Header().Set("Last-Modified", modtime.UTC().Format(http.TimeFormat))
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}
	
		mimeType := mime.TypeByExtension(filepath.Ext(static.Name))
		if mimeType != "" {
			w.Header().Set("Content-Type", mimeType)
		}
		w.Header().Set("Last-Modified", modtime.UTC().Format(http.TimeFormat))
		w.Write([]byte(static.Content))
	}
}

// HTTPHandler handler http request
//eg:on file system is :/res/js/a.js and request is /res/js/a.js
//http.Handle("/res/",res.Assest.HttpHandler("/"))

//eg:on file system is :/res/js/a.js and request is /js/a.js
//http.Handle("/js/",res.Assest.HttpHandler("/res/")) 
func (statics *AssestStruct)HTTPHandler(baseDir string)http.Handler{
	return &_assestFileServer{sf:statics,pdir:baseDir}
}



type _assestFileServer struct{
	sf *AssestStruct
	pdir string
}


// ServeHTTP ServeHTTP
func (f *_assestFileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rname:=filepath.ToSlash(filepath.Join(f.pdir,r.URL.Path))
	f.sf.FileHandlerFunc(rname).ServeHTTP(w,r)
}


func _assestGzipBase64decode(data string)string{
  b,_:=base64.StdEncoding.DecodeString(data)
  gr, _:= gzip.NewReader(bytes.NewBuffer(b))
  bs, _ := ioutil.ReadAll(gr)
  return string(bs)
}

func _assestBase64Decode(data string)string{
   b,_:=base64.StdEncoding.DecodeString(data)
   return string(b)
}

// Assest export assests
var Assest = &AssestStruct{
	Files:map[string]*AssestFile{
	   {{range $file := .files}}
	      _assestBase64Decode("{{$file.Name}}"):&AssestFile{
	         Name:_assestBase64Decode("{{$file.Name}}"),
	         Mtime:{{$file.Mtime}},
	         Content:_assestGzipBase64decode("{{$file.Content}}"),
	       },
		{{end}}
	},
}

`))

var demoConf = `
{
  "src":"res",
  "dest":"res/assest.go",
  "package":"res"
}
`
