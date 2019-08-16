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

	glogging "cloud.google.com/go/logging"
	logging "github.com/ipfs/go-log"
	lwriter "github.com/ipfs/go-log/writer"
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

	/*
		if os.Getenv("GOOGLE") != "" && os.Getenv("K_CONFIGURATION") != "" {
			// lwriter.Configure(lwriter.Output(&LogShim{getStackdriverLogger()}))
			// logging.SetLogLevel("dht", "DEBUG")
			logger := getStackdriverLogger()
			logger.Println("test request")
		} else {
			lwriter.Configure(lwriter.Output(&LogShim{log.New(os.Stderr, "", log.Lshortfile)}))
			logging.SetLogLevel("dht", "DEBUG")
		}
	*/

	var lines []string
	var ns []*dhttests.Node
	n := 2

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
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
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

// LogShim reads the IPFS logs and sends them to StackDriver
type LogShim struct {
	logger *log.Logger
}

func (ls *LogShim) Write(p []byte) (n int, err error) {
	// fmt.Println("Jim LogShim Write", len(p))
	// fmt.Printf("Jim LogShim Write %v", string(p))
	ls.logger.Print(string(p))
	return len(p), nil
}

func main() {
	/*
		logfile, err := os.Create("log.out")
		if err != nil {
			panic(err)
		}
		// w := bufio.NewWriter(logfile)
		// lwriter.Configure(lwriter.Output(w))
		lwriter.Configure(lwriter.Output(logfile))
	*/
	// lwriter.Configure(lwriter.Output(os.Stderr))
	// lwriter.WriterGroup.AddWriter(os.Stderr)

	if os.Getenv("GOOGLE") != "" && os.Getenv("K_CONFIGURATION") != "" {
		lwriter.Configure(lwriter.Output(&LogShim{getStackdriverLogger()}))
		logging.SetLogLevel("dht", "DEBUG")
		/*
			logger := log.New(os.Stderr, "", log.Lshortfile)
			logger.Println("started 4")
			logger2 := getStackdriverLogger()
			logger2.Println("hello world 3.1")
		*/
	} else {
		lwriter.Configure(lwriter.Output(&LogShim{log.New(os.Stderr, "", log.Lshortfile)}))
		logging.SetLogLevel("dht", "DEBUG")
	}

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
	line := fmt.Sprintf("%s runtime: %vms", s, td.Nanoseconds()/(1000*1000))
	fmt.Println(line)
	*lines = append(*lines, line)
	return td
}

func getStackdriverLogger() *log.Logger {
	ctx := context.Background()

	// Sets your Google Cloud Platform project ID.
	projectID := "dht-test-249818"

	// Creates a client.
	client, err := glogging.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	// defer client.Close()

	// Sets the name of the log to write to.
	logName := "dht"

	logger := client.Logger(logName).StandardLogger(glogging.Info)

	// Logs "hello world", log entry is visible at
	// Stackdriver Logs.
	logger.Println("hello world 4")
	return logger
}
