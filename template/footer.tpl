
func init(){

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
