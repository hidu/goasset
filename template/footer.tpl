
func init(){

	{{if ne .debug ""}}
	    exeName := filepath.Base(os.Getenv("_"))
        // only enable with go run
        if exeName == "go" || (runtime.GOOS == "windows" && strings.Contains(os.Args[0], "go-build")) {
            flag.BoolVar(&_assetDirect, "goasset_debug_{{.debug}}", false, "for debug,read asset direct")
        }
	{{end}}

    // asset_include(footer_fn.go)

	asset:= &assetFiles{Files: map[string]*assetFile{}}
	Asset=asset

	{{range $idx,$file := .files}}
	{
	    // {{$idx}} mtime: {{$file.Mtime|ts2str}} , size: {{$file.Size}}
	    fileName:={{$file.Name|printf "%q"}}
	    contentBase64:=`{{$file.Content|long_txt_fmt}}`

		contentGz:=_assetBase64Decode(contentBase64,fileName)
		oneFile:=&assetFile{
			name:fileName,
			mtime:time.Unix({{$file.Mtime}},0),
			content:_assetGzipDecode(contentGz,fileName),
			contentGzip:contentGz,
		}
		asset.Files[fileName]=oneFile
	}
	{{end}}
}
