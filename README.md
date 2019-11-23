goasset
========
go assets tool (V 1.0 20191123)

## 1.Install
```
go get -u github.com/hidu/goasset
```

## 2.Usage:

### 2.1 Cmd
```
 goasset  [-src=resource] [-dest=resource/asset.go] [-package=resource] [-debug=xxx] [asset.json]
```
> note:  
> assets(eg: .css、.js、image files) in resource dir will packed into go source file:`resource/asset.go`

Using it with `go generate` cmd:
```
//go:generate goasset -src template -package internal -dest internal/asset.go
```



> debug:
>> 1: goasset -debug=abc asset.json
>> 2: go run youapp/main.go -goasset_debug_abc

### 2.2 Config File(`asset.json`)
#### a. only one dir: 
```
{
  "src":"res",
  "dest":"serve/asset.go",
  "package":"serve"
}
```

####  b. many dirs:

```
{
  "src":"res|res2",
  "dest":"serve/asset.go",
  "package":"serve"
}
```

### 2.3 Read Asset

#### a: HTTP Handle Support
```
    http.HandleFunc("/index.html", res.Asset.FileHandlerFunc("res/index.html"))
    http.Handle("/res/", res.Asset.HttpHandler("/"))
    
    http.Handle("/js/",res.Asset.HTTPHandler("/res/"))
    
    http.Handle("/static/",http.StripPrefix("/static/",res.Asset.HTTPHandler("/res/")))
```

默认所有 目录名称为 *private 的不允许通过web访问。

#### b: Read Content Directly
```
    content:=res.Asset.GetContent("res/b.css")
    fmt.Println("b.css content:",content)
    
    names := res.Assest.GetFileNames("/")
```

#### c: Demo 
[the demo main.go](demo/main.go) 

## 3 Helper
### a: File Include
a.txt:
```
// asset_include(b.txt)
this is file a
```
b.txt:
```
this is file b
```

### b: Content Remove Range
c.txt:
```
this is file c
a
b
// asset_remove_start()
c 这里的内容将被删除(忽略)掉
c 这里的内容将被删除(忽略)掉
// asset_remove_end()
```

### d: Content Remove Above
d.txt:
```
this is file d
//这里的内容将被删除(忽略)掉
//这里的内容将被删除(忽略)掉
// asset_remove_above()
a
b
```