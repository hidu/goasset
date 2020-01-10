package internal

// Generated by goasset(1.0 20191123). DO NOT EDIT.
// https://github.com/hidu/goasset/

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

	var _assetGzipDecode = func(data []byte, fileName string) []byte {
		gzipReader, errGzip := gzip.NewReader(bytes.NewBuffer(data))
		if errGzip != nil {
			panic(fmt.Sprintf("[goasset](%s) gzip decode failed,err=%s", fileName, errGzip.Error()))
		}
		defer gzipReader.Close()
		buf, errReader := ioutil.ReadAll(gzipReader)
		if errReader != nil {
			panic(fmt.Sprintf("[goasset](%s) read decode content failed,err=%s", fileName, errReader.Error()))
		}
		return buf
	}

	var _assetBase64Decode = func(txt string, fileName string) []byte {
		txt = strings.ReplaceAll(txt, "\n", "")
		bf, err := base64.StdEncoding.DecodeString(txt)
		if err != nil {
			panic(fmt.Sprintf("[goasset](%s) base64 decode failed,err=%s", fileName, err.Error()))
		}
		return bf
	}

	asset := &assetFiles{Files: map[string]*assetFile{}}
	Asset = asset

	{
		// 0 mtime: 2020-01-10 18:36:22 , size: 5684
		fileName := "/template/asset_tpl.go"
		contentBase64 := `
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
AAA=`

		contentGz := _assetBase64Decode(contentBase64, fileName)
		oneFile := &assetFile{
			name:        fileName,
			mtime:       time.Unix(1578652582, 0),
			content:     _assetGzipDecode(contentGz, fileName),
			contentGzip: contentGz,
		}
		asset.Files[fileName] = oneFile
	}

	{
		// 1 mtime: 2019-10-07 11:50:05 , size: 77
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
		// 2 mtime: 2019-11-23 21:23:27 , size: 912
		fileName := "/template/footer.tpl"
		contentBase64 := `
H4sIAAAAAAAA/5RUzY7bNhA+i08xJbILqVC1QVH0IECH3U26pyZAjPbiGg5tjRQCNOmSVNaxzHcvhpTldYsGiQ+GOJyf74cY1g16C1JLnxcjY9k4yg40QtXi
ZuiB8xBYBgCAB3wndgh1A51UuBf+U/UgHObGVU/oUX/O+ZoXBYPpd3cHRqsvgFpsFMKz9J+gN2AHPafIbm7bNMB7w+F0gtwO2ssdVk/v3y/ixbPUrXl2HG5v
wXkrde+qR6O9kNrR/Hvbu+XrVUktftoMUrW8KGCc59CvU6KvHoxRfwqb366Fc+jfSItbH8vieR1Jr8cxsQ+Bl9AJ5bAE3hkLMVpaFC3EfGhjA34hHUhB1G0I
LMYY+ywspGlPR7l/g1vTIjRAsuet8AKWq80Xj2VUNUqRGBbTBYws649y/wFFi7YEtJYakQ8Urt7hc7rKKdvR+WHoOrSxe1GwjFSein5oQEtFLbO90HKbdztf
LfZWat/lfDnJsMpvXBG7Q5vwdkIqbEu0trlx/IJ1hlO9tdbYvKB5gWUtdmjhArt6VMZhXrBsM3SxKMWJhTSDl6qiwL1S+aVohj7lfg/46NEEfmu0R+2/TmLC
eU3Doh+shs3QsfDSSXr3v/5y7aU/+Mm5r1pJac38iD/gXoktEm9/oIf4l+YlcE5CJZ1IoU0cVy18+1ZvTSt1X6XZi9iGSmetvkul1PibTP4fZaIwLIs96wZu
48dvUqEb438NO7FfJrqrH+fbkbbKPZ2aGIuLxwrdI7yS7aF8RbOJekUfjrLHtIbu7mAcKScE2NGWqOlMWdXvdDx597PzNgQowcnji+uFPOJ5mZ2p1c35lk6n
pBLwm7/ntTc9nuR43Xw85z+m+EkZ3a/9wa+7nQ/hI2NZNpU8Hevmv88lv2pYnoEULMuMRtKmbi4qkouZJqCzGRRJxOOO/EPLQ36lQAjl6yKmTaPqf6+ffEZ4
mf+ygDLrSw7LyPHkcRVdXZ7LVs0EOj6K8+YL7B8AAAD//wEAAP//O6i8jlYGAAA=`

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
		// 3 mtime: 2019-11-02 19:27:22 , size: 869
		fileName := "/template/footer_fn.go"
		contentBase64 := `
H4sIAAAAAAAA/5SRQWvjMBCFz/avmDUEZDA6LXtY8GGzDb3lkBzTUGRrZASKbEbjNk3Jfy+yXac5NDRHjWbefPNemr4ogmcVAvLjyXYPWLcaoQTT+1poxQp2
++qNsQBjHa7VASEwWd/k0we8p0lzst0GlUYqAImiEPwtIZblGl/HLxG7Q3wve2OQBvU8TxNr5qFfJXjromTSKW9rYQ4stx1Zz0Zku6YdQPdiEfJBHfTIa5R1
qAskKhchu7DOOHJF1JLI475zmmg0SHDBlv9dG1DkaVL1Zhga6/EK2/ZsnYyFf86Jy9CMPvXeA0+o9Cd83XpGz7ePmDivzyDknjxUvUnPX5NcqoB/fl9nyUee
krsZZWwrp3qQG+ycqjHezUcuIHvyWQFZFo0afYoOVcM6uWW98nWrrW/kuHs7yMTR2au7XBqFfxTyN85EYz4AAAD//wEAAP//q/NnTeQCAAA=`

		contentGz := _assetBase64Decode(contentBase64, fileName)
		oneFile := &assetFile{
			name:        fileName,
			mtime:       time.Unix(1572694042, 0),
			content:     _assetGzipDecode(contentGz, fileName),
			contentGzip: contentGz,
		}
		asset.Files[fileName] = oneFile
	}

	{
		// 4 mtime: 2019-10-09 11:24:12 , size: 112
		fileName := "/template/header.tpl"
		contentBase64 := `
H4sIAAAAAAAA/ypITM5OTE9VqK5W0YOya2u5uPT1FdxT81KLEktSUxSSKhXS8xOLi1NLNEDKylKLijPz82prNfUUXPwV/PxDFFxdPEP0QJoySkoKiq309dMz
SzJKk/SS83P1MzJTSvWh+vW5AAAAAP//AQAA//9V9qlxcAAAAA==`

		contentGz := _assetBase64Decode(contentBase64, fileName)
		oneFile := &assetFile{
			name:        fileName,
			mtime:       time.Unix(1570591452, 0),
			content:     _assetGzipDecode(contentGz, fileName),
			contentGzip: contentGz,
		}
		asset.Files[fileName] = oneFile
	}

	{
		// 5 mtime: 2019-11-02 18:37:45 , size: 3889
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
		// 6 mtime: 2019-11-02 18:26:12 , size: 2944
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
