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


```
	http.HandleFunc("/index.html", res.Assest.FileHandlerFunc("res/index.html"))
	http.Handle("/res/", res.Assest.HttpHandler("/"))
	
	content:=res.Assest.GetContent("res/b.css")
	fmt.Println("b.css content:",content)

	names := res.Assest.GetFileNames("/")
```


[the demo main.go](demo/main.go) 