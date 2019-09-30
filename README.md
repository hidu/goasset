goasset
========
go assets tool

install
```
go get -u github.com/hidu/goasset
```

useage
```
 goasset  [-src=res] [-dest=demo] [-package=res] [asset.json]
```
output file is `assest.go` in asset dir  

goasset conf is: `assest.json`:
```
{
  "src":"res",
  "dest":"serve/asset.go",
  "package":"serve"
}
```

or

```
{
  "src":"res|res2",
  "dest":"serve/asset.go",
  "package":"serve"
}
```


```
    http.HandleFunc("/index.html", res.Asset.FileHandlerFunc("res/index.html"))
    http.Handle("/res/", res.Asset.HttpHandler("/"))
    
    http.Handle("/js/",res.Asset.HTTPHandler("/res/"))
    
    http.Handle("/static/",http.StripPrefix("/static/",res.Asset.HTTPHandler("/res/")))
    
    content:=res.Asset.GetContent("res/b.css")
    fmt.Println("b.css content:",content)
    
    names := res.Assest.GetFileNames("/")
```


[the demo main.go](demo/main.go) 