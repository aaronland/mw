package main

import (
	_ "bytes"
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

	bib_root := filepath.Join(abs_root, "bibliography")
	log.Println(bib_root)

	// mw2020
	// mw97
	// mwa2014
	// mwf2015
	mw_re, err := regexp.Compile(`.*\/(mw(?:[a-z])?)(\d{2,4}).*`)

	if err != nil {
		log.Fatalf("Failed to compile MW regexp, %v", err)
	}
	
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

		bib_id := q.Get("bib")
		
		bib_rsp, err := http.Get(href)

		if err != nil {
			log.Fatalf("Failed to retrieve %s, %v", href, err)
		}

		defer bib_rsp.Body.Close()

		bib_body, err := io.ReadAll(bib_rsp.Body)

		if err != nil {
			log.Fatalf("Failed to read body for %s, %v", href, err)
		}

		bib_doc := soup.HTMLParse(string(bib_body))
		bib_content := bib_doc.Find("div", "id", "primary")

		bib_links := bib_content.FindAll("a")

		for _, link := range bib_links {
			
			bib_href := link.Attrs()["href"]

			if !mw_re.MatchString(bib_href){
				continue
			}

			if !strings.HasPrefix(bib_href, "https"){

			}
			
			m := mw_re.FindStringSubmatch(bib_href)

			conf := fmt.Sprintf("%s-%s", m[1], m[2])
			log.Println(conf, bib_href)

			paper_fname := filepath.Base(bib_href)
			paper_fname = fmt.Sprintf("%s-%s.html", bib_id, paper_fname)

			paper_root := filepath.Join(abs_root, "papers")
			paper_root = filepath.Join(paper_root, conf)

			paper_path := filepath.Join(paper_root, paper_fname)
			pdf_path := strings.Replace(paper_path, ".html", ".pdf", 1)

			err = mw.FetchPDF(path_webster, bib_href, pdf_path)

			if err != nil {
				log.Printf("Failed to derive PDF for %s, %v", bib_href, err)
			}

			continue
			
			// log.Println(paper_path)

			paper_rsp, err := http.Get(bib_href)

			if err != nil {
				log.Printf("Failed to retrieve %s (%s), %v", bib_href, bib_id, err)
				continue
			}

			defer paper_rsp.Body.Close()
			
			err = mw.WriteData(paper_path, paper_rsp.Body)

			if err != nil {
				log.Fatalf("Failed to write data for %s (%s), %v", bib_href, bib_id, err)
			}
		}
			
	}
}
