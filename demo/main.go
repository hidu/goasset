package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/hidu/goasset/demo/res"
)

var port = flag.Int("port", 8080, "http server port")

func main() {
	flag.Parse()

	http.HandleFunc("/index.html", res.Asset.FileHandlerFunc("res/index.html"))
	http.Handle("/res/", res.Asset.HTTPHandler("/"))
	http.Handle("/js/", res.Asset.HTTPHandler("/res/"))

	http.Handle("/js2/", http.StripPrefix("/js2/", res.Asset.HTTPHandler("/res2/")))

	http.Handle("/static/", http.StripPrefix("/static/", res.Asset.HTTPHandler("/res/")))

	addr := fmt.Sprintf(":%d", *port)

	content := res.Asset.GetContent("res/b.css")
	fmt.Println("b.css content:", content)

	names := res.Asset.GetFileNames("/")
	fmt.Println("fileNames of /", names)

	names_0 := res.Asset.GetFileNames("")
	fmt.Println("fileNames of ", names_0)

	names_1 := res.Asset.GetFileNames("/res/js/")
	fmt.Println("fileNames of /res/js/", names_1)

	fmt.Println("pls visit http://" + addr + "/index.html")

	http.ListenAndServe(addr, nil)
}
