package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	dhttests "github.com/jimpick/dht-test-cloud-run/dht"
)

func handler(w http.ResponseWriter, r *http.Request) {
	log.Print("Home page request.")
	body, err := ioutil.ReadFile("static/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(body)
}

func testHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	log.Print("Test request.")

	var lines []string
	var ns []*dhttests.Node
	n := 10

	timed(&lines, "setup", func() {
		ns = nodes(n)
	})
	for i := 0; i < n; i++ {
		fmt.Printf("%v %v\n", i, ns[i].Host.ID())
		line := fmt.Sprintf("%v %v", i, ns[i].Host.ID())
		lines = append(lines, line)
	}

	dsch := make(chan time.Duration, n)
	for i := 1; i < n; i++ {
		go func(i int) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			var err error
			d := timed(&lines, fmt.Sprintf("n0 -> n%d", i), func() {
				_, err = ns[0].DHT.FindPeer(ctx, ns[i].Host.ID())
			})
			if err != nil {
				line := fmt.Sprintf("n0 failed to find n%d. %v", i, err)
				fmt.Println(line)
				lines = append(lines, line)
			}
			dsch <- d
		}(i)
	}
	for i := 1; i < n; i++ {
		<-dsch
	}
	for i := 0; i < n; i++ {
		ns[i].Host.Close()
	}

	lines = append(lines, "")
	io.WriteString(w, strings.Join(lines, "\n"))
}

func main() {
	http.HandleFunc("/", handler)
	http.HandleFunc("/test", testHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8099"
	}

	log.Printf("Web server started on port: %s\n", port)
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
			defer func() {
				if r := recover(); r != nil {
					fmt.Println("Recovered", r)
				}
			}()
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

func timed(lines *[]string, s string, f func()) time.Duration {
	t1 := time.Now()
	f()
	td := time.Since(t1)
	line := fmt.Sprintf("%s runtime: %vms", s, td.Nanoseconds() / (1000*1000))
	fmt.Println(line)
	*lines = append(*lines, line)
	return td
}
