package main

import (
        "fmt"
	"io"
        "io/ioutil"
        "log"
        "net/http"
        "os"
)

func handler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadFile("static/index.html")
        if err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
        }
        w.Write(body)
}

func testHandler(w http.ResponseWriter, r *http.Request) {
	if (r.Method == "POST") {
		log.Print("Test request.")
		io.WriteString(w, "Hello from /test!\n")
	} else {
		http.Error(w, "404 not found.", http.StatusNotFound)
	}
}

func main() {
        log.Print("Hello world sample started.")

        http.HandleFunc("/", handler)
        http.HandleFunc("/test", testHandler)

        port := os.Getenv("PORT")
        if port == "" {
                port = "8080"
        }

        log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
