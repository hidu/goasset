package main

import (
	"flag"
	"fmt"
	"github.com/hidu/goassest/demo/res"
	"net/http"
)

var port = flag.Int("port", 8080, "http server port")

func main() {
	flag.Parse()

	http.HandleFunc("/index.html", res.Assest.FileHandlerFunc("res/index.html"))
	http.Handle("/res/", res.Assest.HTTPHandler("/"))
	http.Handle("/js/",res.Assest.HTTPHandler("/res/"))
	
	
	http.Handle("/static/",http.StripPrefix("/static/",res.Assest.HTTPHandler("/res/")))
	
	addr := fmt.Sprintf("127.0.0.1:%d", *port)

	content := res.Assest.GetContent("res/b.css")
	fmt.Println("b.css content:", content)

	names := res.Assest.GetFileNames("/")
	fmt.Println("fileNames of /", names)

	names_0 := res.Assest.GetFileNames("")
	fmt.Println("fileNames of ", names_0)
	
	names_1 := res.Assest.GetFileNames("/res/js/")
	fmt.Println("fileNames of /res/js/", names_1)

	fmt.Println("pls visit http://" + addr + "/index.html")

	http.ListenAndServe(addr, nil)
}
