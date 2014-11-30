package main

import (
	"./res"
	"flag"
	"fmt"
	"net/http"
)

var debug = flag.Bool("debug", false, "use assest direct?")
var port = flag.Int("port", 8080, "http server port")

func main() {
	flag.Parse()
	if *debug {
		res.Assest.Direct = true
	}
	http.HandleFunc("/index.html", res.Assest.FileHandlerFunc("res/index.html"))
	http.Handle("/res/", res.Assest.HttpHandler("/"))
	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	fmt.Println("pls visit http://" + addr + "/index.html")
	http.ListenAndServe(addr, nil)
}
