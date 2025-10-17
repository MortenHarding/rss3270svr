// rss3270srv.go
package main

import (
	"context"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
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
	Link  string `xml:"link"`
}

/*************** Config ***************/
const (
	httpTimeout  = 10 * time.Second
	maxHeadlines = 18 // fits 24x80 with header/footer
)

var feedURLs = readRssUrlFile("rssfeed.url")
var feedURL = feedURLs[0]
var listenAddr = ""

/*************** Main ***************/
func main() {
	//Define command line arguments
	port := flag.String("port", "7300", "Listen on port")
	flag.Parse()
	listenAddr = ":" + *port

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

		// Accept Enter; PF3/Clear exit and PF4 new url.
		// Place cursor at row 22, col 34 (inside input field)
		pfkeys := []go3270.AID{go3270.AIDEnter, go3270.AIDPF4}
		exitkeys := []go3270.AID{go3270.AIDPF3, go3270.AIDClear}

		headlines, err := fetchHeadlines(feedURL, maxHeadlines)
		if err != nil {
			headlines = []string{fmt.Sprintf("Error fetching feed: %v", err)}
		}

		screen := buildScreen(headlines)

		resp, err := go3270.HandleScreen(screen, nil, nil, pfkeys, exitkeys, "", 22, 70, conn)
		if err != nil {
			log.Printf("HandleScreen: %v", err)
			return
		}
		switch resp.AID {
		case go3270.AIDPF3:
			return
		case go3270.AIDPF4:
			// Ask for a new URL
			fieldValues := make(map[string]string)

			response, err := go3270.HandleScreen(changeURLscreen(), nil, fieldValues, pfkeys, exitkeys, "", 22, 46, conn)
			if err != nil {
				log.Printf("HandleScreen: %v", err)
				return
			}

			if response.AID == go3270.AIDPF3 {
				// Exit
				break
			}
			fieldValues = response.Values
			if fieldValues["choice"] != "" {
				ch := fieldValues["choice"]
				var i int
				if _, err := fmt.Sscanf(ch, "%2d", &i); err == nil {
					fmt.Println(i)
				}
				feedURL = feedURLs[i]
			}
			if fieldValues["feedURL"] != "" {
				feedURL = fieldValues["feedURL"]
			}

		case go3270.AIDClear:
			return
		}
		if strings.EqualFold(strings.TrimSpace(resp.Values["cmd"]), "q") {
			return
		}
	}
}

/*************** Presentation ***************/
func changeURLscreen() go3270.Screen {
	var fields []go3270.Field

	title := " Change RSS URL Feed"
	header := padCenter(title, 79)

	fields = append(fields,
		go3270.Field{Row: 0, Col: 0, Content: header, Intense: true},
		go3270.Field{Row: 1, Col: 0, Content: strings.Repeat("-", 79)}, // ASCII only
		go3270.Field{Row: 2, Col: 0, Content: "Enter URL: "},
		go3270.Field{Row: 2, Col: 10, Name: "feedURL", Write: true, Highlighting: go3270.Underscore},
		go3270.Field{Row: 2, Col: 79, Autoskip: true}, // field "stop" character
		go3270.Field{Row: 3, Col: 0, Content: "Or select from one of the below URL's"},
	)

	row := 5

	for i, url := range feedURLs {
		for _, line := range wrap80(fmt.Sprintf("%2d. %s", i, url), 80) {
			if row >= 20 { // leave space for footer/input
				break
			}
			fields = append(fields, go3270.Field{Row: row, Col: 0, Content: line})
			row++
		}
		if row >= 20 {
			break
		}
	}

	fields = append(fields,
		go3270.Field{Row: 20, Col: 0, Intense: true, Color: go3270.Red, Name: "errormsg"}, // a blank field for error messages
		go3270.Field{Row: 21, Col: 0, Content: strings.Repeat("-", 80)},                   // ASCII only
		go3270.Field{Row: 22, Col: 0, Content: "Press Enter to save, PF3 Exit, # of new url:"},
		go3270.Field{Row: 22, Col: 45, Write: true, Name: "choice", Content: ""},
	)

	return go3270.Screen(fields)
}

func buildScreen(headlines []string) go3270.Screen {
	var fields []go3270.Field

	now := time.Now().UTC().Format("15:04 UTC")
	title := " RSS Feed"
	header := padCenter(title, 80)
	sub := padCenter(feedURL+"  -  Updated: "+now, 80)

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
		go3270.Field{Row: 22, Col: 0, Content: "Command (Enter=refresh, q+Enter=quit, PF3/Clear=exit, PF4=RSS url):"},
		// Input field starts at col 33; cursor set to (22,34) to be inside the unprotected field
		go3270.Field{Row: 22, Col: 69, Write: true, Name: "cmd", Content: ""},
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
		rpl := replaceUnhandledChar(t)
		if rpl != "" {
			out = append(out, rpl)
			if len(out) >= limit {
				break
			}
		}
		//add the url link for the item to the output
		//l := strings.TrimSpace(it.Link)
		//if l != "" {
		//	out = append(out, l)
		//	if len(out) >= limit {
		//		break
		//	}
		//}
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

func replaceUnhandledChar(s string) string {
	//Define characters that must be replaced
	r := strings.NewReplacer(
		"å", "aa",
		"ø", "oe",
		"æ", "ae",
		"Å", "AA",
		"Ø", "OE",
		"Æ", "AE",
		"–", "-",
		"’", "'",
		"‘", "'",
		"ö", "oe",
		"ä", "ae",
		"ü", "ue",
	)

	line := r.Replace(s)

	return line
}

func readRssUrlFile(filename string) []string {
	content, err := os.ReadFile(filename)
	lines := strings.Split(string(content), "\n")
	if err != nil {
		//do something
	}

	return lines

}
