goassest
========
go assets tool

install
```
go get -u github.com/hidu/goassest
```

useage
```
 goassest  [-src=res] [-dest=demo] [-package=res] [assest.json]
```
output file is `assest.go` in assest dir  

assest conf is: `assest.json`:
```
{
  "src":"res",
  "dest":"serve/assest.go",
  "package":"serve"
}
```

or

```
{
  "src":"res|res2",
  "dest":"serve/assest.go",
  "package":"serve"
}
```


```
    http.HandleFunc("/index.html", res.Assest.FileHandlerFunc("res/index.html"))
    http.Handle("/res/", res.Assest.HttpHandler("/"))
    
    http.Handle("/js/",res.Assest.HTTPHandler("/res/"))
    
    http.Handle("/static/",http.StripPrefix("/static/",res.Assest.HTTPHandler("/res/")))
    
    content:=res.Assest.GetContent("res/b.css")
    fmt.Println("b.css content:",content)
    
    names := res.Assest.GetFileNames("/")
```


[the demo main.go](demo/main.go) 