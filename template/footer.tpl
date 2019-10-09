
func init(){
	asset:= &assetFiles{Files: map[string]*assetFile{}}
	Asset=asset
	{{range $file := .files}}
	{
	    fileName:={{$file.Name|printf "%q"}}
		contentGz:=[]byte({{$file.Content|printf "%q"}})
		oneFile:=&assetFile{
			name:fileName,
			mtime:time.Unix({{$file.Mtime}},0),
			content:_assetGzipDecode(contentGz),
			contentGzip:contentGz,
		}
		asset.Files[fileName]=oneFile
	}
	{{end}}
}
