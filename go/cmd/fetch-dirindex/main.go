package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	mw "github.com/aaronland/mw/go"
	"github.com/anaskhan96/soup"
)

func main() {

	var sessions_url string
	var dest string
	var path_webster string

	flag.StringVar(&sessions_url, "sessions-url", "", "The root URL containing session information for a MW conference.")
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

	papers_root := filepath.Join(abs_root, "papers")

	rsp, err := http.Get(sessions_url)

	if err != nil {
		log.Fatalf("Failed to retrieve %s, %v", sessions_url, err)
	}

	defer rsp.Body.Close()

	body, err := io.ReadAll(rsp.Body)

	if err != nil {
		log.Fatalf("Failed to read body from %s, %v", sessions_url, err)
	}

	re_paper, err := regexp.Compile(`^[a-z\-]+\/$`)

	if err != nil {
		log.Fatalf("Failed to compile paper regepx, %v", err)
	}

	doc := soup.HTMLParse(string(body))

	links := doc.FindAll("a")

	for _, link := range links {

		href := link.Attrs()["href"]

		if !re_paper.MatchString(href) {
			continue
		}

		u, _ := url.Parse(sessions_url)
		u.Path = filepath.Join(u.Path, filepath.Join(href, "index.html"))

		paper_uri := u.String()

		r, err := mw.FetchContentWithId(paper_uri, "primary")

		if err != nil {
			log.Fatalf("Failed to fetch paper for %s, %w", err)
		}

		fname := fmt.Sprintf("%s.html", filepath.Base(href))

		path := filepath.Join(papers_root, fname)
		path_pdf := strings.Replace(path, ".html", ".pdf", 1)

		err = mw.WriteData(path, r)

		if err != nil {
			log.Fatalf("Failed to write %s, %v", path, err)
		}

		err = mw.FetchPDF(path_webster, paper_uri, path_pdf)

		if err != nil {
			log.Fatalf("Failed to fetch PDF for %s, %v", path, err)
		}
	}
}
