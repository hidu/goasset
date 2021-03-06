// Generated by goasset(1.0 20200425) or "go generate" . DO NOT EDIT.
// https://github.com/hidu/goasset/

package internal

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
		var errWrote error
		if len(gzipContent) > 0 && strings.Contains(req.Header.Get("Accept-Encoding"), "gzip") {
			writer.Header().Set("Content-Encoding", "gzip")
			_, errWrote = writer.Write(gzipContent)
		} else {
			_, errWrote = writer.Write(file.Content())
		}

		if errWrote != nil {
			log.Printf("[wf] wrote %q with error:%s\n", name, errWrote)
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

func newAssetHelper() *assetHelper {
	helper := &assetHelper{}

	helper.Regs = make(map[string]*regexp.Regexp)
	helper.Regs["remove_above"] = regexp.MustCompile(`[\S\s]*?//\s*asset_remove_above\(\s*\)`)
	helper.Regs["remove"] = regexp.MustCompile(`//\s*asset_remove_start\(\s*\)[\S\s]*?//\s*asset_remove_end\(\s*\)`)
	helper.Regs["include"] = regexp.MustCompile(`//\s*asset_include\(([^)]+?)\)`)

	helper.RegisterFn("remove_above", helper.RemoveAbove)
	helper.RegisterFn("include", helper.Include)
	helper.RegisterFn("remove", helper.Remove)

	return helper
}

type assetHelperFn func(fileName string, content []byte) ([]byte, error)

type assetHelper struct {
	Fns  []map[string]assetHelperFn
	Regs map[string]*regexp.Regexp
}

// RegisterFn 注册helper方法
func (h *assetHelper) RegisterFn(name string, fn assetHelperFn) {
	h.Fns = append(h.Fns, map[string]assetHelperFn{name: fn})
}

// Execute 执行所有的helper方法
func (h *assetHelper) Execute(fileAbsPath string, content []byte, skipFnName string) (contentNew []byte, err error) {
	contentNew = make([]byte, len(content))
	copy(contentNew, content)

	for _, fnInfo := range h.Fns {
		for name, fn := range fnInfo {
			if name == skipFnName {
				continue
			}
			contentNew, err = fn(fileAbsPath, contentNew)
			if err != nil {
				return nil, fmt.Errorf("%s,current file is: %s", err.Error(), fileAbsPath)
			}
		}
	}

	return contentNew, nil
}

// RemoveAbove 删除在此标记之前的内容
// eg: \/\/ asset_remove_above()
func (h *assetHelper) RemoveAbove(fileAbsPath string, content []byte) (contentNew []byte, err error) {
	contentNew = h.Regs["remove_above"].ReplaceAll(content, []byte(""))
	return contentNew, nil
}

// Remove remove 方法, 删除指定区间里的内容
// eg: \/\/asset_remove_start() 中间的内容被删除 \/\/ asset_remove_end()
func (h *assetHelper) Remove(fileAbsPath string, content []byte) (contentNew []byte, err error) {
	contentNew = h.Regs["remove"].ReplaceAll(content, []byte(""))
	return contentNew, nil
}

func (h *assetHelper) include(fileAPath string, content []byte, includeFiles map[string]map[string]bool) (contentNew []byte, err error) {

	fileAPath = filepath.Clean(fileAPath)
	includeFiles[fileAPath] = make(map[string]bool)

	contentNew = h.Regs["include"].ReplaceAllFunc(content, func(matchData []byte) []byte {
		idx := bytes.Index(matchData, []byte("("))
		name := bytes.TrimSpace(matchData[idx+1 : len(matchData)-1])
		if len(name) == 0 {
			err = fmt.Errorf("asset_include with empty param")
			return []byte(err.Error())
		}
		fileBPath := filepath.Join(filepath.Dir(fileAPath), string(name))

		if bFiles, hasB := includeFiles[fileBPath]; hasB {
			if _, hasA := bFiles[fileAPath]; hasA {
				err = fmt.Errorf("asset_include error: cyclic include,%s include(%s)", fileAPath, string(name))
				return []byte(err.Error())
			}
		}
		includeFiles[fileAPath][fileBPath] = true

		includeFiles[fileBPath] = make(map[string]bool)

		bContent, errRead := ioutil.ReadFile(fileBPath)
		if errRead != nil {
			err = errRead
			return []byte(err.Error())
		}

		b1Content, errB1 := h.Execute(fileBPath, bContent, "include")
		if errB1 != nil {
			err = errB1
			return []byte(err.Error())
		}

		cContent, errInclude := h.include(fileBPath, b1Content, includeFiles)
		if errInclude != nil {
			err = errInclude
			return []byte(err.Error())
		}
		return cContent
	})

	if err != nil {
		return nil, err
	}
	return contentNew, nil
}

// Include 将另外一个资源文件包含到当前文件里
// eg: \/\/ asset_include(a.tpl)
func (h *assetHelper) Include(fileAPath string, content []byte) (contentNew []byte, err error) {
	// 用于检查循环include
	includeFiles := make(map[string]map[string]bool)
	return h.include(fileAPath, content, includeFiles)
}

// ---------------------------helper.go--------finish-------------------------//

// Asset export assets
var Asset AssetFiles

func init() {

	// nolint
	var _assetGzipDecode = func(data []byte, fileName string) []byte {
		gzipReader, errGzip := gzip.NewReader(bytes.NewBuffer(data))
		if errGzip != nil {
			panic(fmt.Sprintf("[goasset] gzip decode failed,file=%q, err=%s", fileName, errGzip.Error()))
		}
		defer gzipReader.Close()
		buf, errReader := ioutil.ReadAll(gzipReader)
		if errReader != nil {
			panic(fmt.Sprintf("[goasset] read decode content failed, file=%q err=%s", fileName, errReader.Error()))
		}
		return buf
	}

	// nolint
	var _assetBase64Decode = func(txt string, fileName string) []byte {
		txt = strings.ReplaceAll(txt, "\n", "")
		bf, err := base64.StdEncoding.DecodeString(txt)
		if err != nil {
			panic(fmt.Sprintf("[goasset](%s) base64 decode failed, err=%s", fileName, err.Error()))
		}
		return bf
	}

	asset := &assetFiles{Files: map[string]*assetFile{}}
	Asset = asset

	{
		// 0 mtime: 2020-04-25 21:32:05 , size: 577
		fileName := "/template/asset_test.tpl"
		contentBase64 := `
H4sIAAAAAAAA/2yQMW/bMBCFZ96vuBIOIBUCuRv1UMBu0CUZ6i32QMsnmqlMKuQpbcHyvxdUhU6d+IB377t3BNAaH8lTNExXvPxCG0xKxE3OG/VOMbngS2kx
RJQ2oF1HJSrcP+PT8xEP+69HVTE35ilttbaOb/NF9eGub+4665WoASbTfzeWsLJXXQoAgLtPITI2ICRTYuetBCEjDSP1XGUKkSW0ADDMvscjJf681GT8uCbU
scUMwgbG7Q4XVz0Sf3EjPZk7pUZq2YKoJPVtdD01NnCHlde4Dl/ReW7xEsKIGRARhYjEc/RoA7+486f6vJ7/WqUF8cN43u5ezomj8zaDyDkabwk37vqz2wxu
pNpEVZFKAVGTOS+GqpV+T9F5HlA+vMlSugogf62TBYQb8MP6AWpPNB3eZjMuleve5VTB6hBjiEMj/3PuSeqTbHcn//DeYc0sUnb/EHVNAfgDAAD//wEAAP//
KChZ4QMCAAA=`

		contentGz := _assetBase64Decode(contentBase64, fileName)
		oneFile := &assetFile{
			name:        fileName,
			mtime:       time.Unix(1587821525, 0),
			content:     _assetGzipDecode(contentGz, fileName),
			contentGzip: contentGz,
		}
		asset.Files[fileName] = oneFile
	}

	{
		// 1 mtime: 2020-04-25 09:56:58 , size: 5958
		fileName := "/template/asset_tpl.go"
		contentBase64 := `
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`

		contentGz := _assetBase64Decode(contentBase64, fileName)
		oneFile := &assetFile{
			name:        fileName,
			mtime:       time.Unix(1587779818, 0),
			content:     _assetGzipDecode(contentGz, fileName),
			contentGzip: contentGz,
		}
		asset.Files[fileName] = oneFile
	}

	{
		// 2 mtime: 2019-10-07 11:50:05 , size: 77
		fileName := "/template/demo_config.json"
		contentBase64 := `
H4sIAAAAAAAA/6rmUlBQKi5KVrJSKkotzi8tSk7VV9IBCaakFpcgiyYWF6eW6KXnQ2QLEpOzE9NTkRQocdUCAAAA//8BAAD//2RKZxFNAAAA`

		contentGz := _assetBase64Decode(contentBase64, fileName)
		oneFile := &assetFile{
			name:        fileName,
			mtime:       time.Unix(1570420205, 0),
			content:     _assetGzipDecode(contentGz, fileName),
			contentGzip: contentGz,
		}
		asset.Files[fileName] = oneFile
	}

	{
		// 3 mtime: 2019-11-23 21:23:27 , size: 912
		fileName := "/template/footer.tpl"
		contentBase64 := `
H4sIAAAAAAAA/4xUS4/bNhA+i79iSmQXUqHKQVH0IECH3U26pyZAjPbiGg5tjRQCNOmQVNaxzP9eDPVwnHaD9WEhDufxPbjDmk7vQGrp06xnLOl72YBGKGrc
di1wHgJLAADwiO/EHqGsoJEKD8J/Ku6Fw9S44hE96i8p3/AsYzD+FgswWn0F1GKrEJ6k/wStAdvpOUU2c9uqAt4aDuczpLbTXu6xeHz/fhkvnqSuzZPjcHsL
zlupW1c8GO2F1I7m39nWrV6vc2rxy7aTquZZBv08h36NEm1xb4z6W9j0diOcQ/9GWtz5WBbPm0h60/cD+xB4Do1QDnPgjbEQo7lFUUPMhzo24BfSgRREXYfA
YoyxxQK0UVJ79kVYGOY+nuThDe5MjVABGZDWwgtYrbdfPeZR3yjKwDUbL6BnSXuShw8oarQ5oLXUiByhcPEOn4arlLIdne+7pkEbu2cZS0jvseinCrRU1DI5
CC13abP3xfJgpfZNylejIOvYGeoBayOkwjoneNXN5wigunH8AnjGVLy11tg0o6GBJTU2aOGCvXhQxmGasWTbNbFoiBMVaTovVUGBO6XSS9GMf8x9KYNo18hg
Z7RH7ScmMFJ5hskI9pqLRd9ZDduuYeH/3aX/it9/u/bXH/3o5g/tpbRqfuIf8KDEDkkGf6Rn+o/mOXBOug2ykWDbOK5Y+vqt3pla6rYYZi9jGyqdpXupaOmN
y8bG35n/jFTPiBQ1YklsWlZwGz/+kApdH/+WsBeH1cB3/fN829PSuaNTFWNxL1mhW4RXsj7mr2g2cS/ow1F2P2ypxQL6nnJCgD0tkZLOlFX8Scezd786b0OA
HJw8fXO9lCecdt1EraymWzqdB5mA33yet+L4oAbLy+rjlP8wxM/K6Hbjj37T7H0IHxlLkrHk8VRW/30v6VXDfAKSsSQxGkmbsrqoSDYmmoDOZlBkIB5X6F9a
HtMrBULIX2cxbRxVfr+T0hnhZf63BZRZXnJYQo4PHhfR1dVUtq5G0PFRTIsxsH8BAAD//wEAAP//QMoxJ3UGAAA=`

		contentGz := _assetBase64Decode(contentBase64, fileName)
		oneFile := &assetFile{
			name:        fileName,
			mtime:       time.Unix(1574515407, 0),
			content:     _assetGzipDecode(contentGz, fileName),
			contentGzip: contentGz,
		}
		asset.Files[fileName] = oneFile
	}

	{
		// 4 mtime: 2020-04-04 10:46:41 , size: 900
		fileName := "/template/footer_fn.go"
		contentBase64 := `
H4sIAAAAAAAA/4yRQYvbMBCFz9avmBoCMhjlUnoo+NC0obcckmMaimyNjECRXGncplny3xfZjkOWzZKjRjNv3vuGseUSnLfGEfsrA/yWMSL9PJvuBzZeIVSg
e9dwJUnC/lD/JyxBG4sbeUSIFIxri+kDXljWnk23RakwlIAhJCH4WkEqiw3+G7946o7pveq1xjCoFwXLjJ6HPlXgjE2SWSedabg+kth1wTjSPN+3fjB6GJRB
jV61NBZVmexViz+DgWoR85vh2ZNYh+ADL9LSC8sUagxw8y6+Wx+RFyyrez0MjfUUxfiejBWp8M1afhua/U+9zyYIKNU1QeMdoaNrEpiiPEgymb3PEpD64KDu
Nbu8f92VjPjl8/196UTTNT88b2qrpnoUW+ysbDBhoBOVkP9yeQl5nriN2BKwelgndqTWrvHKuFaMu3eDTBqd0T0LjS9iMQm/Of4DVA8gJUavAAAA//8BAAD/
//K8LM8DAwAA`

		contentGz := _assetBase64Decode(contentBase64, fileName)
		oneFile := &assetFile{
			name:        fileName,
			mtime:       time.Unix(1585968401, 0),
			content:     _assetGzipDecode(contentGz, fileName),
			contentGzip: contentGz,
		}
		asset.Files[fileName] = oneFile
	}

	{
		// 5 mtime: 2020-03-30 17:08:57 , size: 131
		fileName := "/template/header.tpl"
		contentBase64 := `
H4sIAAAAAAAA/yzMsQrCMBCH8T1P8ac46JLbnSviYpe+QNoelyD2SnIVpOTdRcj2DR8/Itx55RyMF0xfiIZS2M7HcfIfziXpWusFmtGJQtrawaMf8BxG3PrH
6B0RotlWrkSSLO6Tn/VNMS07NZGc28L8CsL4261rde4HAAD//wEAAP//RsZntIMAAAA=`

		contentGz := _assetBase64Decode(contentBase64, fileName)
		oneFile := &assetFile{
			name:        fileName,
			mtime:       time.Unix(1585559337, 0),
			content:     _assetGzipDecode(contentGz, fileName),
			contentGzip: contentGz,
		}
		asset.Files[fileName] = oneFile
	}

	{
		// 6 mtime: 2019-11-02 18:37:45 , size: 3889
		fileName := "/template/helper.go"
		contentBase64 := `
H4sIAAAAAAAA/7RXX2sbxxd91n6K+xP4x46zlapXBRGsJKIuNBSnb5LqrFYja8lqtOyOYhkj8INjx1RJA20dagJNW7c1CXENSYvt0n4a7dr+FmX+7O7IWtkq
pU/Wzty598y5Z86MNa3VIxYQvLrg+5h+hB0XezqCeTP5hHUt0xa/iiX4vzKzPtCiqdwSXvGhBB3zIdY7plv1qWeTlfq8h1dw32XTuO+isfBq1sOd7iO8bDa6
j3C2DiWQ0Z/0fHq723FtB+sPqrX7Nb8+fyufr/kC17K6rqbX/PkaepCee2rWyWw+NT0qs02viUlzWkWbWE6vOVNJGVrT9ernqH7jFuLp1Hy2T7FXIfo4RwbE
EWx0gQ2i1GURmHjFohhIj5ZkXUrPEHmY9jwix7WBptE1F4MiggoBpiG9ZTv4ntnBIDpvgNUlFBMK1XpjjWIEuvhhAPa8rocmM7GVPYsyvVWID1CtK0Iaq6hl
uNym6ozhzOch2SGE7w6CraHYRbh7Er77Rihfb49pHSlrdKLupkXGN434scgxoCUwXReTps4/DZiGep0lLEKLDJAEeLePrR7FEO78cv79MNzZCF/unO1tzgBT
ruSsLzT8T03ankK8Af5D260QpTcIdBlzD6+C0hbZGrY1JUCe6ijOwSRajhALdNeUdHF5Jp5W14Nlxt0iaXWZe3gmWcEgaFvXMjyAscL5jedlPAvI2C0eAKWS
ug8+xTHapIfZx0CTAxEMtp0StIhKUQzuHl5FMjuL+18JiO3IrFLxxHYMaHVo7i7jpKVn53zD6nkeY5alBNsvwpyf5ZVEkI4MUKqhCNdAywySo6SCJLYTazU+
0BA8eXXx7X7w8iB8ux++2j4/PBqdfBHsPD3b2wy2HgeHJ2wBXilCLV/Lw6Ql6miquOMiMyjnH8uknWrruSXsOqaFFxwnymfIbHo2yyQ0AzEgcoI4FIakKBxu
B4d7wfD04sX7i+1hKj+THq8jGB2/vXjxPo4//+GNSJjCKDvYV/P531P570hMxy4vCAH+Sv+QkRXbwWOeq/xsdLvODHvUMkm1Ej8qrknbudsONkkCBGkZtWQ1
nqinPDB4ZW0Ke/GNrNBXYZdVTCG/ujomtdp3TGrG3RJ/uSHYzT5zJvbt5xZJE/eT+KQFOu9BhltVHP2ZZ3fuu6aFkxVVu9m/UYAid9F4FH1QqLPldouPsyyI
Od6HwpKklSleNPaKgFWbtgF3XLoGrumZnSx3HikIiVAxKSQsifeizHtRVJrxcdcWveBfd2xPaYwhNSIQMtoZ5AZvkwFt0y+zVBPN40XqN0VA5OrLfMECZ+ty
n2+KKeHH1+2eq6sI1prl2FZU3JjzY4XP+SgrjVlcAuObyFzHVeTh01Sp7BFKQL0e1tKCy9dJONO4HckSe94SNpuczW6P2k6OfbJcepxKCiYKVe8wQZmcmUEL
rHZBLV4usNLtnPrMKAvyEpDx8UqQlAvpOMqF2VBYKgj5YhVIVLuKkCSQVbITNFGCVEhycqaTEnmrrKdlBqxfk48H9emAPY9d+1dfbhHC4Ggr+PLnYH93dLwx
On59/ttmePo83N0e/fF7MHwcPH8TPDkK/vwq2HkqBi+2hylvgIgjM0ddZ9qdtTij789yZeXzcPb1wej0WfjjRvjdT8Ffr8+e/WpHxI7dG8VJ3U8cgfifjdzE
5RSju9zrgfY3AAAA//8BAAD//zR/kcTEDgAA`

		contentGz := _assetBase64Decode(contentBase64, fileName)
		oneFile := &assetFile{
			name:        fileName,
			mtime:       time.Unix(1572691065, 0),
			content:     _assetGzipDecode(contentGz, fileName),
			contentGzip: contentGz,
		}
		asset.Files[fileName] = oneFile
	}

	{
		// 7 mtime: 2019-11-02 18:26:12 , size: 2944
		fileName := "/template/helper_test.go"
		contentBase64 := `
H4sIAAAAAAAA/zyQzWrrMBCF1zNPMVcQkEEo+0AW4RJooKtuS6H+GdmmieRKk9il+N2LZNrtOZ/OJwbrlFjw3SAAnJeJWzmQKqHK2WoQVkQY+DpxpMORPM+n
XD+VRFeI4EKksVsMtXXii/Atc7H2PZck0TcCiH0OvdPKh5ni3ZdG77pKmfy2QoDGGeJYJJvOvvAtPPjUhAdrZe1eOElXS71vrCyiDL2+NV/C+tdrL366S5XH
Rle2/h3Jj9fyARB7jjFEp1XmaR5lyEyIB9olVdwbocvCigB/w/8Hbj+2+5w/tRhKEkff68ZldsUVfwAAAP//AQAA///AEt/xSgEAAA==`

		contentGz := _assetBase64Decode(contentBase64, fileName)
		oneFile := &assetFile{
			name:        fileName,
			mtime:       time.Unix(1572690372, 0),
			content:     _assetGzipDecode(contentGz, fileName),
			contentGzip: contentGz,
		}
		asset.Files[fileName] = oneFile
	}

}
