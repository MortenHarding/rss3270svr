// rss3270srv.go
package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	go3270 "github.com/racingmars/go3270"
)

/*************** RSS parsing ***************/
type rss struct {
	Channel struct {
		Title string    `xml:"title"`
		Items []rssItem `xml:"item"`
	} `xml:"channel"`
}
type rssItem struct {
	Title string `xml:"title"`
}

/*************** Config ***************/
const (
	listenAddr   = ":7300" // TN3270 port
	feedURL      = "https://feeds.bbci.co.uk/news/world/rss.xml"
	httpTimeout  = 10 * time.Second
	maxHeadlines = 18 // fits 24x80 with header/footer
)

/*************** Main ***************/
func main() {
	log.Printf("Starting 3270 RSS server on %s ...", listenAddr)
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept: %v", err)
			continue
		}
		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()

	// Negotiate TN3270 (returns two values; ignore the first)
	if _, err := go3270.NegotiateTelnet(conn); err != nil {
		log.Printf("NegotiateTelnet failed: %v", err)
		return
	}

	// Loop: Enter=refresh; 'q'+Enter, PF3, or Clear=exit
	for {
		headlines, err := fetchHeadlines(feedURL, maxHeadlines)
		if err != nil {
			headlines = []string{fmt.Sprintf("Error fetching feed: %v", err)}
		}

		screen := buildScreen(headlines)

		// Accept Enter; PF3/Clear exit. Place cursor at row 22, col 34 (inside input field)
		pfkeys := []go3270.AID{go3270.AIDEnter}
		exitkeys := []go3270.AID{go3270.AIDPF3, go3270.AIDClear}

		resp, err := go3270.HandleScreen(screen, nil, nil, pfkeys, exitkeys, "", 22, 34, conn)
		if err != nil {
			log.Printf("HandleScreen: %v", err)
			return
		}
		if resp.AID == go3270.AIDPF3 || resp.AID == go3270.AIDClear {
			return
		}
		if strings.EqualFold(strings.TrimSpace(resp.Values["cmd"]), "q") {
			return
		}
	}
}

/*************** Presentation ***************/
func buildScreen(headlines []string) go3270.Screen {
	var fields []go3270.Field

	now := time.Now().UTC().Format("2006-01-02 15:04 UTC")
	title := " BBC World Headlines (Enter=refresh, q+Enter=quit, PF3/Clear=exit) "
	header := padCenter(title, 80)
	sub := padCenter("Source: feeds.bbci.co.uk/news/world/rss.xml  -  Updated: "+now, 80)

	fields = append(fields,
		go3270.Field{Row: 0, Col: 0, Content: header, Intense: true},
		go3270.Field{Row: 1, Col: 0, Content: sub},
		go3270.Field{Row: 2, Col: 0, Content: strings.Repeat("-", 80)}, // ASCII only
	)

	row := 3
	for i, h := range headlines {
		for _, line := range wrap80(fmt.Sprintf("%2d. %s", i+1, strings.TrimSpace(h)), 80) {
			if row >= 21 { // leave space for footer/input
				break
			}
			fields = append(fields, go3270.Field{Row: row, Col: 0, Content: line})
			row++
		}
		if row >= 21 {
			break
		}
	}

	fields = append(fields,
		go3270.Field{Row: 21, Col: 0, Content: strings.Repeat("-", 80)}, // ASCII only
		go3270.Field{Row: 22, Col: 0, Content: "Command (Enter=refresh, q=quit):"},
		// Input field starts at col 33; cursor set to (22,34) to be inside the unprotected field
		go3270.Field{Row: 22, Col: 33, Write: true, Name: "cmd", Content: ""},
	)

	return go3270.Screen(fields)
}

/*************** Helpers ***************/
func fetchHeadlines(url string, limit int) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var r rss
	if err := xml.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}
	out := make([]string, 0, limit)
	for _, it := range r.Channel.Items {
		t := strings.TrimSpace(it.Title)
		if t != "" {
			out = append(out, t)
			if len(out) >= limit {
				break
			}
		}
	}
	if len(out) == 0 {
		out = []string{"(No headlines found)"}
	}
	return out, nil
}

func wrap80(s string, width int) []string {
	var lines []string
	s = strings.ReplaceAll(s, "\n", " ")
	for len(s) > width {
		cut := width
		if idx := strings.LastIndex(s[:width], " "); idx > 0 {
			cut = idx
		}
		lines = append(lines, padRight(s[:cut], width))
		s = strings.TrimSpace(s[cut:])
	}
	lines = append(lines, padRight(s, width))
	return lines
}

func padRight(s string, w int) string {
	if len(s) >= w {
		return s[:w]
	}
	return s + strings.Repeat(" ", w-len(s))
}

func padCenter(s string, w int) string {
	if len(s) >= w {
		return s[:w]
	}
	left := (w - len(s)) / 2
	right := w - len(s) - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}
