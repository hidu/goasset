package template

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"strings"
)

// asset_remove_above()

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
