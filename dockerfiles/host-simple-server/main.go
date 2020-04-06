package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	ip := flag.String("ip", "127.0.0.1", "ip addr")
	port := flag.String("port", "8084", "port")
	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, "hello %s\n", time.Now())
	})
	addr := fmt.Sprintf("%s:%s", *ip, *port)
	log.Println("serving on", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
