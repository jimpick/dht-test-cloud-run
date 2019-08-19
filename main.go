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

	"cloud.google.com/go/profiler"
	glogging "cloud.google.com/go/logging"
	logging "github.com/ipfs/go-log"
	lwriter "github.com/ipfs/go-log/writer"
	dhttests "github.com/jimpick/dht-test-cloud-run/dht"
	gologging "github.com/whyrusleeping/go-logging"
)

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

func setupLogging(logID string, trace string) {
	var stdLogger *log.Logger
	if glogClient != nil {
		var logger *glogging.Logger
		if trace != "" {
			labelsOpt := glogging.CommonLabels(map[string]string{
				"trace": trace,
			})
			logger = glogClient.Logger(logID, labelsOpt)
		}
		stdLogger = logger.StandardLogger(glogging.Info)
	} else {
		stdLogger = log.New(os.Stderr, "", log.Lshortfile)
	}
	lwriter.Configure(lwriter.Output(&LogShim{stdLogger}))
	logging.SetAllLoggers(gologging.ERROR)
	logging.SetLogLevel("dht", "DEBUG")
}

var glogClient *glogging.Client

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

	trace := r.Header.Get("X-Cloud-Trace-Context")
	log.Print("Test request ", trace)
	setupLogging("dht", trace)

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

	if trace != "" {
		lines = append(lines, "https://console.cloud.google.com/logs/viewer?"+
			"authuser=1&organizationId=882980064372&project=dht-test-249818&"+
			"minLogLevel=0&expandAll=false&"+
			"timestamp=2019-08-17T04:04:27.313542520Z&"+
			"customFacets=&limitCustomFacetWidth=true&"+
			"dateRangeStart=2019-08-17T03:32:36.762Z&"+
			"interval=PT1H&resource=project%2Fproject_id%2Fdht-test-249818&"+
			"scrollTimestamp=2019-08-17T04:31:26.141887534Z&"+
			"logName=projects%2Fdht-test-249818%2Flogs%2Fdht&"+
			"advancedFilter=resource.type%3D%22project%22%0A"+
			"resource.labels.project_id%3D%22dht-test-249818%22%0A"+
			"logName%3D%22projects%2Fdht-test-249818%2Flogs%2Fdht%22%0A"+
			"labels.trace%3D%22"+
			trace+
			// "330438a0b7b19d7cdf7a94dd0a0713b5%2F3255878610758515717;o%3D1" +
			"%22&dateRangeEnd=2019-08-17T04:32:36.762Z")
	}
	lines = append(lines, "")
	io.WriteString(w, strings.Join(lines, "\n"))
}

func main() {
	if os.Getenv("GOOGLE") != "" && os.Getenv("K_CONFIGURATION") != "" {
		glogClient = getStackdriverLogClient()
		// Profiler initialization, best done as early as possible.
		if err := profiler.Start(profiler.Config{
			Service:        "dht-test-2",
			ServiceVersion: "1.0.0",
			DebugLogging: true,
			// MutexProfiling: true,
			// ProjectID must be set if not running on GCP.
			// ProjectID: "my-project",
		}); err != nil {
			fmt.Println("Profiler error", err)
			// TODO: Handle error.
		}
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

func getStackdriverLogClient() *glogging.Client {
	ctx := context.Background()

	// Sets your Google Cloud Platform project ID.
	projectID := "dht-test-249818"

	// Creates a client.
	client, err := glogging.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	return client
}
