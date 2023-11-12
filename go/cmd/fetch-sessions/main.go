package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/anaskhan96/soup"
)

type fetchContentOptions struct {
	IsPre2006 bool
}

type derivePaperURIOptions struct {
	IsPre2010 bool
}

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

	// START OF write sessions to disk

	br := bytes.NewReader(body)

	path_sessions := filepath.Join(abs_root, "sessions.html")

	err = writeData(path_sessions, br)

	if err != nil {
		log.Fatalf("Failed to write %s, %v", path_sessions, err)
	}

	// END OF write sessions to disk

	doc := soup.HTMLParse(string(body))

	links := doc.FindAll("a")

	for _, link := range links {

		href := link.Attrs()["href"]

		href_ok := false

		fetch_opts := fetchContentOptions{}

		paper_uri_opts := derivePaperURIOptions{}

		//     -1998: TBD
		// 1999-2010: abstracts
		// 2011-2012: programs
		// 2013-    : SNFU

		if strings.HasPrefix(href, "programs") {
			href_ok = true
		} else if strings.HasPrefix(href, "../abstracts") {
			href_ok = true
			fetch_opts.IsPre2006 = true
			paper_uri_opts.IsPre2010 = true
		} else {
			//
		}

		if !href_ok {
			continue
		}

		// We have validated this above
		u, _ := url.Parse(sessions_url)

		if err != nil {
			log.Fatalf("Failed to parse %s, %v", sessions_url)
		}

		root := filepath.Dir(u.Path)
		fname := filepath.Base(href)

		u.Path = filepath.Join(root, href)
		url := u.String()

		log.Println("FETCH", url)
		sr, err := fetchContent(url, fetch_opts)

		if err != nil {
			log.Printf("Failed to retrieve session data for %s, %v", url, err)
			continue
		}

		path_s := filepath.Join(abs_root, "sessions")
		path_s = filepath.Join(path_s, fname)

		err = writeData(path_s, sr)

		if err != nil {
			log.Fatalf("Failed to write sessions data for %s, %v", url, err)
		}

		sr.Seek(0, 0)

		has_paper, paper_uri, err := derivePaperURI(sessions_url, sr, paper_uri_opts)

		if err != nil {
			log.Fatalf("Failed to determine is %s has paper, %v", url, err)
		}

		if !has_paper {
			continue
		}

		log.Println("FETCH PAPER", paper_uri)

		pr, err := fetchPaper(paper_uri, fetch_opts)

		path_p := filepath.Join(abs_root, "papers")
		path_p = filepath.Join(path_p, fname)

		err = writeData(path_p, pr)

		if err != nil {
			log.Fatalf("Failed to write paper data for %s, %v", url, err)
		}

		path_pdf := strings.Replace(path_p, ".html", ".pdf", 1)

		log.Printf("FETCH '%s' SAVE '%s\n", paper_uri, path_pdf)

		err = fetchPaperPDF(path_webster, paper_uri, path_pdf)

		if err != nil {
			log.Fatalf("Failed to fetch for PDF for %s, %v", paper_uri, err)
		}
	}
}

func fetchContent(url string, opts fetchContentOptions) (io.ReadSeeker, error) {

	rsp, err := soup.Get(url)

	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve %s, %w", url, err)
	}

	doc := soup.HTMLParse(rsp)

	var r io.ReadSeeker

	if opts.IsPre2006 {
		r = strings.NewReader(doc.HTML())
	} else {

		ev := doc.Find("div", "id", "main-content")
		r = strings.NewReader(ev.HTML())
	}

	return r, nil
}

func derivePaperURI(sessions_url string, r io.Reader, opts derivePaperURIOptions) (bool, string, error) {

	body, err := io.ReadAll(r)

	if err != nil {
		return false, "", fmt.Errorf("Failed to read sessions data, %w", err)
	}

	doc := soup.HTMLParse(string(body))
	links := doc.FindAll("a")

	paper_uri := ""

	for _, link := range links {

		href := link.Attrs()["href"]

		if !strings.HasPrefix(href, "../papers") {
			continue
		}

		paper_uri = href
		break
	}

	if paper_uri == "" {
		return false, "", nil
	}

	paper_uri = strings.Replace(paper_uri, "../", "", 1)

	u, _ := url.Parse(sessions_url)
	root := filepath.Dir(u.Path)

	if opts.IsPre2010 {
		root = filepath.Dir(root)
	}

	u.Path = filepath.Join(root, paper_uri)
	p_url := u.String()

	return true, p_url, nil
}

func fetchPaper(paper_uri string, opts fetchContentOptions) (io.Reader, error) {
	return fetchContent(paper_uri, opts)
}

func fetchPaperPDF(path_webster string, paper_uri string, dest_uri string) error {

	// log.Printf("%s %s %s\n", path_webster, paper_uri, dest_uri)

	cmd := exec.Command(path_webster, paper_uri, dest_uri)
	return cmd.Run()
}

func writeData(path string, r io.Reader) error {

	root := filepath.Dir(path)

	_, err := os.Stat(root)

	if err != nil {

		if !os.IsNotExist(err) {
			return fmt.Errorf("Failed to stat %s, %w", root, err)
		}

		err = os.MkdirAll(root, 0755)

		if err != nil {
			return fmt.Errorf("Failed to create tree for %s, %w", root, err)
		}
	}

	wr, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)

	if err != nil {
		return fmt.Errorf("Failed to open %s for writing, %w", path, err)
	}

	_, err = io.Copy(wr, r)

	if err != nil {
		return fmt.Errorf("Failed to write body to %s, %w", path, err)
	}

	err = wr.Close()

	if err != nil {
		return fmt.Errorf("Failed to close %s after writing, %w", path, err)
	}

	return nil
}
