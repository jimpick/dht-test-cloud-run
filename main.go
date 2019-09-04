package main

import (
	"context"
	"encoding/json"
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

	// dhttests "github.com/jimpick/dht-test-cloud-run/dht"
	dhttests "github.com/jimpick/dht-test-cloud-run/dhtnode"
	// gologging "github.com/whyrusleeping/go-logging"
	cid "github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
	dht "github.com/libp2p/go-libp2p-kad-dht"
)

// LogShim reads the IPFS logs and sends them to StackDriver
type LogShim struct {
	glogger *glogging.Logger
}

func (ls *LogShim) Write(p []byte) (n int, err error) {
	line := strings.Split(string(p), " ")
	date := line[0]
	time := line[1]
	level := line[2]
	subsystem := line[3]
	payload := strings.Join(line[2:], " ")
	if ls.glogger == nil {
		// fmt.Print(string(p))
	} else {
		ls.glogger.Log(glogging.Entry{
			Labels: map[string]string{
				"type":      "log",
				"date":      date,
				"time":      time,
				"level":     level,
				"subsystem": subsystem,
			},
			Payload: payload,
		})
	}
	return len(p), nil
}

// EventShim gets IPFS events and sends them to StackDriver
type EventShim struct {
	glogger *glogging.Logger
}

func (es *EventShim) Write(p []byte) (n int, err error) {
	if es.glogger == nil {
		// fmt.Print("Event: ", string(p))
	} else {
		if err == nil {
			es.glogger.Log(glogging.Entry{
				Labels: map[string]string{
					"type": "event",
				},
				Payload: json.RawMessage(p),
			})
		}
	}
	return len(p), nil
}

// Close closes
func (es *EventShim) Close() error {
	return nil
}

func setupLogging(trace string) *glogging.Logger {
	var glogger *glogging.Logger
	if glogClient != nil {
		labelsOpt := glogging.CommonLabels(map[string]string{
			"trace": trace,
		})
		glogger = glogClient.Logger("dht", labelsOpt)
	}
	lwriter.Configure(lwriter.Output(&LogShim{glogger}))
	lwriter.WriterGroup.Close()
	lwriter.WriterGroup = lwriter.NewMirrorWriter()
	lwriter.WriterGroup.AddWriter(&EventShim{glogger})
	// logging.SetAllLoggers(gologging.ERROR)
	// logging.SetLogLevel("dht", "DEBUG")
	logging.SetDebugLogging()
	return glogger
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
	glogger := setupLogging(trace)

	var lines []string
	var ns []*dhttests.Node
	n := 1

	timed(&lines, "setup", func() {
		ns = nodes(n)
	})
	for i := 0; i < n; i++ {
		fmt.Printf("%v %v\n", i, ns[i].Host.ID())
		line := fmt.Sprintf("%v %v", i, ns[i].Host.ID())
		lines = append(lines, line)
	}

	targetContent, err := cid.Decode("QmYdGGwQXpaBGTVWXqMFVXUP2CZhtsV29jxPkRm54ArAdT") // Filecoin proof
	// targetContent, err := cid.Decode("QmX3g1tW8cZX7fQD9VPFx1zpWNtGDAqX8bse8Hg2Nak9G3") // protocol.ai
	// targetContent, err := cid.Decode("Qmbsb7MMCveTEdfeTjCJPssJmPwbN2jfGyUDSo8oPfk9mB") // ipfs.io
	// targetContent, err := cid.Decode("QmbynpFBGvNygCJQPVKxb8hDmXCumtBPLdiTbSxA8vTE8x") // docs.ipfs.io
	// targetPeer, err := peer.IDB58Decode("QmScdku7gc3VvfZZvT8kHU77bt6bnH3PnGXkyFRZ17g9EG") // home.jimpick.com
	// targetPeer, err := peer.IDB58Decode("QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ") // mars.i.ipfs.io
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		var err error
		/*
			d := timed(&lines, fmt.Sprintf("n0 -> target %v", targetPeer), func() {
				_, err = ns[0].DHT.FindPeer(ctx, targetPeer)
			})
		*/
		var addrInfos []peer.AddrInfo
		d := timed(&lines, fmt.Sprintf("n0 -> providers for cid %v", targetContent), func() {
			addrInfos, err = ns[0].DHT.FindProviders(ctx, targetContent)
		})
		line := fmt.Sprintf("duration: %v", d)
		lines = append(lines, line)
		fmt.Println(line)
		if err == nil {
			numResults := len(addrInfos)
			line := fmt.Sprintf("Number of results: %v", numResults)
			lines = append(lines, line)
			fmt.Println(line)
			for _, addrInfo := range addrInfos {
				line := fmt.Sprintf("  Addr: %v", addrInfo.ID)
				lines = append(lines, line)
				fmt.Println(line)
				for _, maddr := range addrInfo.Addrs {
					line := fmt.Sprintf("    %v", maddr)
					lines = append(lines, line)
					fmt.Println(line)
				}
			}
			if glogger != nil {
				result := map[string]interface{}{
					"randomPeerId":     ns[0].Host.ID(),
					"targetContentCid": targetContent,
					"duration":         d,
					"numResults":       numResults,
					"addrInfos":        addrInfos,
				}
				jsonData, err := json.Marshal(result)
				if err == nil {
					fmt.Println("JSON:", string(jsonData))
					glogger.Log(glogging.Entry{
						Labels: map[string]string{
							"type": "result",
						},
						Payload: json.RawMessage(jsonData),
					})
				}
			}
		} else {
			line := fmt.Sprintf("n0 failed to find %v. %v", targetContent, err)
			lines = append(lines, line)
			fmt.Println(line)
		}
	} else {
		lines = append(lines, fmt.Sprintf("Peer ID err: %v", err))
	}
	for i := 0; i < n; i++ {
		ns[i].Host.Close()
	}
	setupLogging(trace + " finished") // Continue logging until next request

	if trace != "" {
		lines = append(lines, "https://console.cloud.google.com/logs/viewer?"+
			"authuser=1&organizationId=882980064372&project=dht-test-249818&"+
			"minLogLevel=0&expandAll=false&"+
			"customFacets=&limitCustomFacetWidth=true&"+
			"interval=PT1H&"+
			"resource=project%2Fproject_id%2Fdht-test-249818&"+
			"logName=projects%2Fdht-test-249818%2Flogs%2Fdht&"+
			"advancedFilter=resource.type%3D%22project%22%0A"+
			"resource.labels.project_id%3D%22dht-test-249818%22%0A"+
			"labels.trace%3D%22"+
			trace+"%22")
	}
	lines = append(lines, "")
	io.WriteString(w, strings.Join(lines, "\n"))
}

func main() {
	dht.AlphaValue = 10
	fmt.Println("AlphaValue:", dht.AlphaValue)

	if os.Getenv("GOOGLE") != "" && os.Getenv("K_CONFIGURATION") != "" {
		glogClient = getStackdriverLogClient()
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
	cfg := dhttests.DefaultNodeCfg()
	cfg.Bootstrap = dhttests.BootstrapAddrs
	n, err := dhttests.NewNode(cfg)
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
		fmt.Printf("Latency %d ", i) // no newline
		dhttests.PrintLatencyTable(os.Stdout, ns[i].Host)
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
