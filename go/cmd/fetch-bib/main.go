package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	_ "regexp"
	"strings"

	mw "github.com/aaronland/mw/go"
	"github.com/anaskhan96/soup"
)

func main() {

	var sessions_url string
	var dest string
	var path_webster string

	flag.StringVar(&sessions_url, "sessions-url", "https://www.museweb.net/bibliography/?by=all", "The root URL containing session information for a MW conference.")
	flag.StringVar(&dest, "destination", "", "The folder where MW session and paper data will be saved to..")
	flag.StringVar(&path_webster, "webster", "", "The path to a local instance of the https://github.com/aaronland/webster-cli binary is stored; used to produce PDF files for papers")

	flag.Parse()

	abs_root, err := filepath.Abs(dest)

	if err != nil {
		log.Fatalf("Failed to derive absolute path for %s, %v", dest, err)
	}

	_, err = url.Parse(sessions_url)

	if err != nil {
		log.Fatalf("Invalid sessions URL, %v", err)
	}

	// papers_root := filepath.Join(abs_root, "papers")

	bib_root := filepath.Join(abs_root, "bibliography")
	
	//
	
	rsp, err := http.Get(sessions_url)

	if err != nil {
		log.Fatalf("Failed to retrieve %s, %v", sessions_url, err)
	}

	defer rsp.Body.Close()

	body, err := io.ReadAll(rsp.Body)

	if err != nil {
		log.Fatalf("Failed to read body from %s, %v", sessions_url, err)
	}

	br := bytes.NewReader(body)


	path_bib := filepath.Join(bib_root, "bibliography.html")

	err = mw.WriteData(path_bib, br)

	if err != nil {
		log.Fatalf("Failed to write bibliography, %v", err)
	}
	
	doc := soup.HTMLParse(string(body))

	links := doc.FindAll("a")

	for _, link := range links {

		href := link.Attrs()["href"]

		if !strings.HasPrefix(href, "https://www.museweb.net/bibliography/?bib="){
			continue
		}

		u, err := url.Parse(href)

		if err != nil {
			log.Fatalf("Failed to parse %s, %v", href, err)
		}

		q := u.Query()
		id := q.Get("bib")

		fname := fmt.Sprintf("%s.html", id)

		bib_r, err := mw.FetchContentWithId(href, "primary")

		if err != nil {
			log.Fatalf("Failed to get %s, %v", href, err)
		}

		bib_path := filepath.Join(bib_root, fname)

		err = mw.WriteData(bib_path, bib_r)

		if err != nil {
			log.Fatalf("Failed to write data for %s (%s), %v", bib_path, href, err)
		}
	}
}
