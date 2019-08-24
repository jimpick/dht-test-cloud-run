package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/logging"
	"cloud.google.com/go/logging/logadmin"
	"google.golang.org/api/iterator"
)

var (
	client    *logadmin.Client
	projectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
)

type parsedEntry struct {
	Entry     *logging.Entry
	DateTime  string
	TraceID   string
	StartTime string
}

func main() {
	// This example demonstrates how to iterate through items a page at a time
	// even if each successive page is fetched by a different process. It is a
	// complete web server that displays pages of log entries. To run it as a
	// standalone program, rename both the package and this function to "main".
	ctx := context.Background()
	if projectID == "" {
		log.Fatal("GOOGLE_CLOUD_PROJECT missing env")
	}
	var err error
	client, err = logadmin.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("creating logging client: %v", err)
	}

	http.HandleFunc("/entries", handleEntries)
	http.HandleFunc("/vis-data/", handleVisData)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8011"
	}

	log.Print("listening on ", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}

var pageTemplate = template.Must(template.New("").Parse(`
<table>
  {{range $i, $v := .Entries}}
    <tr>
        <td>
            {{$i}} {{.DateTime}}
            <a href="/vis-data/{{.TraceID}}">Vis Data</a>
            <a href="https://console.cloud.google.com/logs/viewer?authuser=0&project=dht-test-249818&minLogLevel=0&expandAll=false&customFacets=&limitCustomFacetWidth=true&interval=JUMP_TO_TIME&resource=cloud_run_revision%2Fservice_name%2Fdht-test-2&dateRangeStart={{.StartTime}}&timestamp={{.StartTime}}&advancedFilter=resource.type%3D%22cloud_run_revision%22%0Aresource.labels.service_name%3D%22dht-test-2%22%0AlogName%3D%22projects%2Fdht-test-249818%2Flogs%2Frun.googleapis.com%252Fstderr%22%20OR%20logName%3D%22projects%2Fdht-test-249818%2Flogs%2Frun.googleapis.com%252Fstdout%22%0Alabels.instanceId%3D%22{{.Entry.Labels.instanceId}}%22&dateRangeEnd=2019-08-24T21:18:43.000Z&scrollTimestamp={{.StartTime}}" target="_blank">Instance</a>
            <a href="https://console.cloud.google.com/logs/viewer?authuser=0&project=dht-test-249818&minLogLevel=0&expandAll=false&timestamp=2019-08-24T07:15:45.499000000Z&customFacets=&limitCustomFacetWidth=true&interval=NO_LIMIT&resource=project&scrollTimestamp=2019-08-23T20:46:33.361358546Z&advancedFilter=resource.type%3D%22project%22%0Alabels.trace%3D%22{{.TraceID}}%22%0Alabels.type%3D%22log%22" target="_blank">Logs</a>
            <a href="https://console.cloud.google.com/logs/viewer?authuser=0&project=dht-test-249818&minLogLevel=0&expandAll=false&timestamp=2019-08-24T07:15:45.499000000Z&customFacets=&limitCustomFacetWidth=true&interval=NO_LIMIT&resource=project&scrollTimestamp=2019-08-23T20:46:33.361358546Z&advancedFilter=resource.type%3D%22project%22%0Alabels.trace%3D%22{{.TraceID}}%22%0Alabels.type%3D%22event%22" target="_blank">Events</a>
            {{.TraceID}}
        </td>
    </tr>
  {{end}}
</table>
{{if .Next}}
  <a href="/entries?pageToken={{.Next}}">Next Page</a>
{{end}}
`))

func handleEntries(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	filter := fmt.Sprintf(
		// `resource.type="cloud_run_revision" `+
		// `resource.labels.service_name="%s" `+
		`logName = "projects/%s/logs/run.googleapis.com%%2Fstderr" `+
			`AND "Test request" `+
			`AND timestamp > "2019-08-23T13:10:00-07:00" `+
			`AND timestamp < "2019-08-23T13:50:00-07:00"`,
		projectID)
	it := client.Entries(ctx, logadmin.Filter(filter))
	var entries []*logging.Entry
	nextTok, err := iterator.NewPager(it, 1000, r.URL.Query().Get("pageToken")).NextPage(&entries)
	if err != nil {
		http.Error(w, fmt.Sprintf("problem getting the next page: %v", err), http.StatusInternalServerError)
		return
	}
	parsedEntries := make([]parsedEntry, len(entries))
	for i, entry := range entries {
		var values []string
		if str, ok := entry.Payload.(string); ok {
			values = strings.Split(str, " ")
		} else {
			panic("Wrong type")
		}
		parsedEntries[i] = parsedEntry{
			Entry:     entry,
			DateTime:  values[0] + " " + values[1],
			TraceID:   values[4],
			StartTime: entry.Timestamp.Format(time.RFC3339),
		}
	}

	data := struct {
		Entries []parsedEntry
		Next    string
	}{
		parsedEntries,
		nextTok,
	}
	/*
			for i, v := range data.Entries {
				// fmt.Println("Jim", i, v.TraceID, v.Entry.Labels["instanceId"])
				fmt.Println("Jim", i, v.StartTime)
		    }
	*/
	var buf bytes.Buffer
	if err := pageTemplate.Execute(&buf, data); err != nil {
		http.Error(w, fmt.Sprintf("problem executing page template: %v", err), http.StatusInternalServerError)
	}
	if _, err := buf.WriteTo(w); err != nil {
		log.Printf("writing response: %v", err)
	}
}

func handleVisData(w http.ResponseWriter, r *http.Request) {
	traceID := strings.Replace(r.URL.Path, "/vis-data/", "", 1)
	ctx := context.Background()
	filter := fmt.Sprintf(
		`resource.type="project" `+
			`AND labels.trace="%s" `+
			`AND labels.type="event"`,
		traceID)
	it := client.Entries(ctx, logadmin.Filter(filter))
	for {
		entry, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			panic("Error")
		}
		data, err := json.Marshal(entry.Payload)
		if err == nil {
			fmt.Fprintf(w, "data: %v\n\n", string(data))
		}
	}
}
