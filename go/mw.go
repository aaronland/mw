package mw

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/anaskhan96/soup"
)

func FetchContentWithId(url string, id string) (io.ReadSeeker, error) {

	rsp, err := soup.Get(url)

	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve %s, %w", url, err)
	}

	doc := soup.HTMLParse(rsp)

	ev := doc.Find("div", "id", id)
	r := strings.NewReader(ev.HTML())

	return r, nil
}

func WriteData(path string, r io.Reader) error {

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

func FetchPDF(path_webster string, paper_uri string, dest_uri string) error {

	// log.Printf("%s %s %s\n", path_webster, paper_uri, dest_uri)

	cmd := exec.Command(path_webster, paper_uri, dest_uri)
	return cmd.Run()
}
