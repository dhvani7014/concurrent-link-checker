# Concurrent Link Checker
![Go CI](https://github.com/dhvani7014/concurrent-link-checker/actions/workflows/go.yml/badge.svg)

🔗 **Live demo:** https://concurrent-link-checker.onrender.com

A CLI tool written in Go that crawls a webpage, extracts all its links, and checks them concurrently using a worker-pool pattern (goroutines + channels).

## Features
- Concurrent link checking using a configurable worker pool (goroutines + channels)
- Detects broken links (4xx/5xx status codes and connection errors/timeouts)
- Reports response time for every link checked

## Usage

```bash
go run main.go https://example.com
```

Example output:
```
Fetching https://example.com ...
Found 1 links. Checking with 10 concurrent workers...
[OK]     https://iana.org/domains/example -> status 200 (233ms)
Done. 1 OK, 0 broken, 1 total.
```

## How it works
1. Fetches the given URL and parses the HTML to extract all `<a href="">` links
2. Spins up a pool of worker goroutines
3. Each worker pulls a link off a shared `jobs` channel and checks it with an HTTP GET
4. Results are collected concurrently on a `results` channel and printed as they arrive

## Tech
- Go standard library (`net/http`, `sync`, `os`)
- `golang.org/x/net/html` for HTML parsing


