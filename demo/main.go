package main

import (
	"./res"
	"flag"
	"fmt"
	"net/http"
	"os"
)

var port = flag.Int("port", 8080, "http server port")

func main() {
	flag.Parse()
	http.HandleFunc("/index.html", res.Assest.FileHandlerFunc("res/index.html"))
	http.Handle("/res/", res.Assest.HttpHandler("/"))
	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	fmt.Println("pls visit http://" + addr + "/index.html")
	names := res.Assest.GetFileNames("/")
	fmt.Println(names)
	http.ListenAndServe(addr, nil)
}
