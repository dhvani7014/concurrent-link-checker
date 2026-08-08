package main

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const pageTemplate = `
<!DOCTYPE html>
<html>
<head>
	<title>Concurrent Link Checker</title>
	<style>
		body { font-family: -apple-system, sans-serif; max-width: 900px; margin: 40px auto; padding: 0 20px; background: #0d1117; color: #e6edf3; }
		h1 { color: #58a6ff; }
		form { margin-bottom: 30px; }
		input[type=text] { width: 70%; padding: 10px; font-size: 14px; border-radius: 6px; border: 1px solid #30363d; background: #161b22; color: #e6edf3; }
		button { padding: 10px 20px; font-size: 14px; border-radius: 6px; border: none; background: #238636; color: white; cursor: pointer; }
		button:hover { background: #2ea043; }
		table { width: 100%; border-collapse: collapse; margin-top: 20px; }
		th, td { text-align: left; padding: 8px; border-bottom: 1px solid #30363d; font-size: 13px; }
		.ok { color: #3fb950; }
		.broken { color: #f85149; }
		.summary { margin-top: 15px; font-weight: bold; }
	</style>
</head>
<body>
	<h1>Concurrent Link Checker</h1>
	<form method="POST" action="/check">
		<input type="text" name="url" placeholder="https://example.com" value="{{.SubmittedURL}}" required>
		<button type="submit">Check Links</button>
	</form>

	{{if .Results}}
	<div class="summary">Checked {{.Total}} links — {{.OK}} OK, {{.Broken}} broken</div>
	<table>
		<tr><th>Status</th><th>URL</th><th>Code</th><th>Time</th></tr>
		{{range .Results}}
		<tr>
			<td class="{{if .Broken}}broken{{else}}ok{{end}}">{{if .Broken}}BROKEN{{else}}OK{{end}}</td>
			<td>{{.URL}}</td>
			<td>{{.StatusCode}}</td>
			<td>{{.Duration}}</td>
		</tr>
		{{end}}
	</table>
	{{end}}
</body>
</html>
`

type PageData struct {
	SubmittedURL string
	Results      []DisplayResult
	Total        int
	OK           int
	Broken       int
}

type DisplayResult struct {
	URL        string
	StatusCode int
	Duration   string
	Broken     bool
}

func runServer() {
	tmpl := template.Must(template.New("page").Parse(pageTemplate))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl.Execute(w, PageData{})
	})

	http.HandleFunc("/check", func(w http.ResponseWriter, r *http.Request) {
		targetURL := r.FormValue("url")
		if targetURL == "" {
			tmpl.Execute(w, PageData{})
			return
		}

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(targetURL)
		if err != nil {
			tmpl.Execute(w, PageData{SubmittedURL: targetURL})
			return
		}
		defer resp.Body.Close()

		buf := new(strings.Builder)
		io.Copy(buf, resp.Body)

		links := extractLinks(targetURL, buf.String())

		jobs := make(chan string, len(links))
		results := make(chan Result, len(links))
		var wg sync.WaitGroup

		numWorkers := 10
		for i := 1; i <= numWorkers; i++ {
			wg.Add(1)
			go worker(i, client, jobs, results, &wg)
		}
		for _, link := range links {
			jobs <- link
		}
		close(jobs)

		go func() {
			wg.Wait()
			close(results)
		}()

		var display []DisplayResult
		var okCount, brokenCount int
		for res := range results {
			isBroken := res.Err != nil || res.StatusCode >= 400
			if isBroken {
				brokenCount++
			} else {
				okCount++
			}
			display = append(display, DisplayResult{
				URL:        res.URL,
				StatusCode: res.StatusCode,
				Duration:   res.Duration.String(),
				Broken:     isBroken,
			})
		}

		tmpl.Execute(w, PageData{
			SubmittedURL: targetURL,
			Results:      display,
			Total:        len(display),
			OK:           okCount,
			Broken:       brokenCount,
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Server running at http://localhost:%s\n", port)
	http.ListenAndServe(":"+port, nil)
}