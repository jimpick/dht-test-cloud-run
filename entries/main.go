package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	"cloud.google.com/go/logging"
	"cloud.google.com/go/logging/logadmin"
	"google.golang.org/api/iterator"
)

var (
	client    *logadmin.Client
	projectID = flag.String("project-id", "", "ID of the project to use")
)

type parsedEntry struct {
	Entry    *logging.Entry
	DateTime string
	TraceID  string
}

func main() {
	// This example demonstrates how to iterate through items a page at a time
	// even if each successive page is fetched by a different process. It is a
	// complete web server that displays pages of log entries. To run it as a
	// standalone program, rename both the package and this function to "main".
	ctx := context.Background()
	flag.Parse()
	if *projectID == "" {
		log.Fatal("-project-id missing")
	}
	var err error
	client, err = logadmin.NewClient(ctx, *projectID)
	if err != nil {
		log.Fatalf("creating logging client: %v", err)
	}

	http.HandleFunc("/entries", handleEntries)
	log.Print("listening on 8011")
	log.Fatal(http.ListenAndServe(":8011", nil))
}

var pageTemplate = template.Must(template.New("").Parse(`
<table>
  {{range .Entries}}
    <tr>
        <td>
            {{.DateTime}} {{.TraceID}}
            <a href="https://console.cloud.google.com/logs/viewer?authuser=0&project=dht-test-249818&minLogLevel=0&expandAll=false&timestamp=2019-08-24T07:15:45.499000000Z&customFacets=&limitCustomFacetWidth=true&interval=NO_LIMIT&resource=project&scrollTimestamp=2019-08-23T20:46:33.361358546Z&advancedFilter=resource.type%3D%22project%22%0Alabels.trace%3D%22{{.TraceID}}%22%0Alabels.type%3D%22log%22">Logs</a>
            <a href="https://console.cloud.google.com/logs/viewer?authuser=0&project=dht-test-249818&minLogLevel=0&expandAll=false&timestamp=2019-08-24T07:15:45.499000000Z&customFacets=&limitCustomFacetWidth=true&interval=NO_LIMIT&resource=project&scrollTimestamp=2019-08-23T20:46:33.361358546Z&advancedFilter=resource.type%3D%22project%22%0Alabels.trace%3D%22{{.TraceID}}%22%0Alabels.type%3D%22event%22">Events</a>
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
		*projectID)
	it := client.Entries(ctx, logadmin.Filter(filter))
	var entries []*logging.Entry
	nextTok, err := iterator.NewPager(it, 20, r.URL.Query().Get("pageToken")).NextPage(&entries)
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
			Entry:    entry,
			DateTime: values[0] + " " + values[1],
			TraceID:  values[4],
		}
	}

	data := struct {
		Entries []parsedEntry
		Next    string
	}{
		parsedEntries,
		nextTok,
	}
	for i, v := range data.Entries {
		fmt.Println("Jim", i, v.TraceID)
	}
	var buf bytes.Buffer
	if err := pageTemplate.Execute(&buf, data); err != nil {
		http.Error(w, fmt.Sprintf("problem executing page template: %v", err), http.StatusInternalServerError)
	}
	if _, err := buf.WriteTo(w); err != nil {
		log.Printf("writing response: %v", err)
	}
}
