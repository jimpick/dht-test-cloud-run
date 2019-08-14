package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	dhttests "github.com/jimpick/dht-test-cloud-run/dht"
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
	if r.Method == "POST" {
		log.Print("Test request.")

		var ns []*dhttests.Node
		n := 5

		timed("setup", func() {
			ns = nodes(n)
		})
		for i := 1; i < n; i++ {
			fmt.Printf("%v %v\n", i, ns[i].Host.ID())
		}
		io.WriteString(w, "Hello from /test!\n")
	} else {
		http.Error(w, "404 not found.", http.StatusNotFound)
	}
}

func main() {
	log.Print("Web server started.")

	http.HandleFunc("/", handler)
	http.HandleFunc("/test", testHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}

func node() *dhttests.Node {
	n, err := dhttests.NewNode()
	if err != nil {
		panic(err)
	}

	time.Sleep(time.Second)
	return n
}

func nodes(n int) []*dhttests.Node {
	ch := make(chan *dhttests.Node, n)
	for i := 0; i < n; i++ {
		go func() {
			ch <- node()
		}()
	}

	// wait for bootstrapping & latency gathering.
	time.Sleep(6 * time.Second)

	ns := make([]*dhttests.Node, n)
	for i := 0; i < n; i++ {
		ns[i] = <-ch
		fmt.Printf("%d ", i) // no newline
		// PrintLatencyTable(ns[i].Host)
	}
	fmt.Printf("bootstrapped %d nodes\n", n)
	return ns
}

func timed(s string, f func()) time.Duration {
	t1 := time.Now()
	f()
	td := time.Since(t1)
	fmt.Printf("%s runtime: %v\n", s, td)
	return td
}
