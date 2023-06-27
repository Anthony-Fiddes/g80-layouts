package main

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
)

const baseURL = "https://my.glove80.com/api/layouts/v1/"

type Layout struct {
	Metadata struct {
		UUID               string   `json:"uuid"`
		Date               int      `json:"date"`
		Creator            string   `json:"creator"`
		ParentUUID         string   `json:"parent_uuid"`
		FirmwareAPIVersion string   `json:"firmware_api_version"`
		Title              string   `json:"title"`
		Notes              string   `json:"notes"`
		Tags               []string `json:"tags"`
		Unlisted           bool     `json:"unlisted"`
		Deleted            bool     `json:"deleted"`
		Compiled           bool     `json:"compiled"`
		Searchable         bool     `json:"searchable"`
	} `json:"layout_meta"`
	// Don't know what these two do yet
	Config        any `json:"config"`
	CompilerInput any `json:"compiler_input"`
}

func main() {
	if len(os.Args) > 2 {
		log.Fatalf("%s only takes 1 argument at most (a comma separated list of tags to search for)", os.Args[0])
	}
	searchUrl, err := url.Parse(baseURL)
	if err != nil {
		panic(err)
	}
	if len(os.Args) == 2 {
		tags := os.Args[1]
		searchUrl.Query().Add("tags", tags)
	}
	resp, err := http.Get(searchUrl.String())
	io.Copy(os.Stdout, resp.Body)
}
