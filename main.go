package main

import (
	"fmt"
	"net/http"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

// Result holds the outcome of checking a single link.
type Result struct {
	URL        string
	StatusCode int
	Err        error
	Duration   time.Duration
}

// extractLinks parses HTML and returns all href values from <a> tags.
func extractLinks(baseURL string, body string) []string {
	var links []string
	tokenizer := html.NewTokenizer(strings.NewReader(body))

	for {
		tt := tokenizer.Next()
		if tt == html.ErrorToken {
			break
		}
		if tt == html.StartTagToken {
			token := tokenizer.Token()
			if token.Data == "a" {
				for _, attr := range token.Attr {
					if attr.Key == "href" {
						link := resolveURL(baseURL, attr.Val)
						if link != "" {
							links = append(links, link)
						}
					}
				}
			}
		}
	}
	return links
}

// resolveURL turns relative links into absolute ones and skips junk (mailto, #, etc).
func resolveURL(base, href string) string {
	href = strings.TrimSpace(href)
	if href == "" || strings.HasPrefix(href, "#") ||
		strings.HasPrefix(href, "mailto:") || strings.HasPrefix(href, "javascript:") {
		return ""
	}
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	if strings.HasPrefix(href, "//") {
		return "https:" + href
	}
	baseTrimmed := strings.TrimSuffix(base, "/")
	if strings.HasPrefix(href, "/") {
		parts := strings.SplitN(baseTrimmed, "/", 4)
		if len(parts) >= 3 {
			return parts[0] + "//" + parts[2] + href
		}
	}
	return baseTrimmed + "/" + href
}

// checkLink performs an HTTP GET and reports status + timing.
func checkLink(client *http.Client, url string) Result {
	start := time.Now()
	resp, err := client.Get(url)
	duration := time.Since(start)

	if err != nil {
		return Result{URL: url, Err: err, Duration: duration}
	}
	defer resp.Body.Close()

	return Result{URL: url, StatusCode: resp.StatusCode, Duration: duration}
}

// worker pulls URLs off the jobs channel and sends Results to the results channel.
func worker(id int, client *http.Client, jobs <-chan string, results chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()
	for url := range jobs {
		results <- checkLink(client, url)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <url>")
		fmt.Println("Example: go run main.go https://example.com")
		os.Exit(1)
	}

	startURL := os.Args[1]
	numWorkers := 10

	client := &http.Client{Timeout: 10 * time.Second}

	fmt.Printf("Fetching %s ...\n", startURL)
	resp, err := client.Get(startURL)
	if err != nil {
		fmt.Printf("Failed to fetch start URL: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	buf := new(strings.Builder)
	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		fmt.Printf("Failed to read body: %v\n", err)
		os.Exit(1)
	}

	links := extractLinks(startURL, buf.String())
	fmt.Printf("Found %d links. Checking with %d concurrent workers...\n\n", len(links), numWorkers)

	jobs := make(chan string, len(links))
	results := make(chan Result, len(links))
	var wg sync.WaitGroup

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

	var broken, ok int
	for res := range results {
		if res.Err != nil {
			fmt.Printf("[BROKEN] %s -> error: %v\n", res.URL, res.Err)
			broken++
		} else if res.StatusCode >= 400 {
			fmt.Printf("[BROKEN] %s -> status %d (%v)\n", res.URL, res.StatusCode, res.Duration)
			broken++
		} else {
			fmt.Printf("[OK]     %s -> status %d (%v)\n", res.URL, res.StatusCode, res.Duration)
			ok++
		}
	}

	fmt.Printf("\nDone. %d OK, %d broken, %d total.\n", ok, broken, ok+broken)
}