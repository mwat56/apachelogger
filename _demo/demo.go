package main

import (
	"io"
	"log"
	"net/http"
	"os"

	"github.com/mwat56/apachelogger"
)

// `myHandler()` is a dummy for demonstration purposes.
func myHandler(aWriter http.ResponseWriter, aRequest *http.Request) {
	io.WriteString(aWriter, "Hello world!")
} // myHandler()

func main() {
	// the filename should be taken from the commandline or a config file:
	logfile := "/dev/stderr"

	pageHandler := http.NewServeMux()
	pageHandler.HandleFunc("/", myHandler)

	server := http.Server{
		Addr:    "127.0.0.1:8080",
		Handler: apachelogger.Wrap(pageHandler, logfile),
	}

	if err := server.ListenAndServe(); nil != err {
		log.Fatalf("%s: %v", os.Args[0], err)
	}
} // main()
