package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"cloud.google.com/go/logging"
	"cloud.google.com/go/logging/logadmin"
	"google.golang.org/api/iterator"
)

var (
	client    *logadmin.Client
	projectID = flag.String("project-id", "", "ID of the project to use")
)

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
    <tr><td>{{.Payload}}</td></tr>
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
	data := struct {
		Entries []*logging.Entry
		Next    string
	}{
		entries,
		nextTok,
	}
	for i, v := range data.Entries {
		fmt.Println("Jim", i, v.Payload, v.Resource.Labels)
	}
	var buf bytes.Buffer
	if err := pageTemplate.Execute(&buf, data); err != nil {
		http.Error(w, fmt.Sprintf("problem executing page template: %v", err), http.StatusInternalServerError)
	}
	if _, err := buf.WriteTo(w); err != nil {
		log.Printf("writing response: %v", err)
	}
}
