/*
 * Copyright(C) 2020 github.com/hidu  All Rights Reserved.
 * Author: hidu (duv123+git@baidu.com)
 * Date: 2020/4/25
 */

package template
// asset_remove_above()

// asset_include(header.tpl)

import (
	"testing"
	"reflect"
	"sort"
)


func TestAsset(t *testing.T) {
	got := Asset.GetFileNames("/")
	sort.Slice(got, func(i, j int) bool {
    		return got[i]<got[j]
    	})
	want:=[]string{
	{{range $idx,$file := .files}}
	    {{$file.Name|printf "%q"}},
	{{end}}
	}
	if !reflect.DeepEqual(got,want) {
		t.Errorf("Asset.GetFileNames(\"/\")=\n%v, want=\n%v",got,want)
	}
}

