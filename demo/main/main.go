package main

import (
	"../"
	"flag"
	"fmt"
	"net/http"
)

var debug = flag.Bool("debug", false, "use assest direct?")
var port = flag.Int("port", 8080, "http server port")

func main() {
	flag.Parse()
	if *debug {
		demo.DebugAssestDir = "../"
	}
	http.Handle("/", demo.Files.HttpHandler("/"))
	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	fmt.Println("pls visit http://" + addr + "/index.html")
	http.ListenAndServe(addr, nil)
}
