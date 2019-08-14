package dhttests

import (
        "fmt"
        "io/ioutil"
        "log"
        "net/http"
        "os"
)

func handler(w http.ResponseWriter, r *http.Request) {
        log.Print("Hello world received a request.")
	body, err := ioutil.ReadFile("static/index.html")
        if err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
        }
        w.Write(body)
}

func main() {
        log.Print("Hello world sample started.")

        http.HandleFunc("/", handler)

        port := os.Getenv("PORT")
        if port == "" {
                port = "8080"
        }

        log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
