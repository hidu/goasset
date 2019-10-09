package template

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
)

// asset_remove_above()

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

func (h *assetHelper) RegisterFn(name string, fn assetHelperFn) {
	h.Fns = append(h.Fns, map[string]assetHelperFn{name: fn})
}

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
