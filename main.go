package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/adrg/xdg"
	"github.com/olekukonko/tablewriter"
)

const (
	baseURL       = "https://my.glove80.com/api/layouts/v1/"
	cacheFileName = "g80-layouts-cache.json"
	cachePerms    = 0644
)

type Layout struct {
	Metadata struct {
		UUID string `json:"uuid"`
		// Date is a unix time stamp
		Date               int64    `json:"date"`
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

func (l Layout) Time() time.Time {
	return time.Unix(int64(l.Metadata.Date), 0)
}

// SemanticHash gives us a way of only showing a layout if it differs from
// another based on any of the fields in the "hash"
func (l Layout) SemanticHash() string {
	return fmt.Sprintf("%s-%s", l.Metadata.Title, l.Metadata.Creator)
}

func (l Layout) AsRow() []string {
	date := l.Time().Format("1/2/06")
	return []string{date, l.Metadata.Title, l.Metadata.Notes, l.Metadata.Creator}
}

var (
	cache     = make(map[string]Layout)
	cachePath = ""
)

func readCache() {
	cacheBytes, err := ioutil.ReadFile(cachePath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Print("Could not read layout cache.")
		panic(err)
	}
	json.Unmarshal(cacheBytes, &cache)
}

func writeCache() {
	cacheBytes, err := json.Marshal(cache)
	if err != nil {
		log.Print("Could not save layout cache.")
		panic(err)
	}
	err = ioutil.WriteFile(cachePath, cacheBytes, cachePerms)
	if err != nil {
		log.Printf("Could not write cache to disk: %s", err)
		return
	}
	log.Print("Successfully wrote cache to disk.")
}

func getLayout(uid string) Layout {
	if _, ok := cache[uid]; ok {
		return cache[uid]
	}

	layoutURL, err := url.Parse(baseURL)
	if err != nil {
		panic(err)
	}
	layoutURL.Path, err = url.JoinPath(layoutURL.Path, uid, "meta")
	if err != nil {
		panic(err)
	}
	log.Printf("Requesting layout: %s", layoutURL.String())
	resp, err := http.Get(layoutURL.String())
	if err != nil {
		panic(err)
	}
	layoutBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	result := Layout{}
	json.Unmarshal(layoutBytes, &result)
	cache[uid] = result
	return result
}

func main() {
	var err error
	cachePath, err = xdg.CacheFile(cacheFileName)
	if err != nil {
		log.Print("Could not find an XDG location to put the cache file")
		// I'd rather panic than mess up someone's home folder lol
		panic(err)
	}

	var (
		debug  bool
		limit  int
		offset int
		redupe bool
	)
	flag.BoolVar(&debug, "debug", false, "Whether to print debug statements")
	flag.BoolVar(&redupe, "redupe", false, "Whether to show layouts with the same title by the same creator")
	flag.IntVar(&limit, "limit", 10, "How many layouts to show")
	flag.IntVar(&offset, "offset", 0, "How many layouts to skip")
	flag.Parse()
	args := flag.Args()
	if len(args) > 1 {
		log.Fatalf("%s only takes 1 argument at most (a comma separated list of tags to search for)", os.Args[0])
	}
	if !debug {
		// This is a script, so we're just going to panic if anything goes
		// wrong. I.e. all logs are for debugging.
		log.Default().SetOutput(io.Discard)
	}

	readCache()
	defer writeCache()

	searchURL, err := url.Parse(baseURL)
	if err != nil {
		panic(err)
	}
	if len(args) == 1 {
		tags := args[0]
		query := url.Values{}
		query.Add("tags", tags)
		searchURL.RawQuery = query.Encode()
	}
	log.Printf("Requesting layout unique IDs: %s", searchURL.String())
	resp, err := http.Get(searchURL.String())
	var uids []string
	uidBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(uidBytes, &uids)
	if err != nil {
		panic(err)
	}

	seenLayouts := make(map[string]struct{})
	var rows [][]string
	for _, uid := range uids[offset : offset+limit] {
		layout := getLayout(uid)
		if redupe {
			rows = append(rows, layout.AsRow())
			continue
		}

		hash := layout.SemanticHash()
		_, exists := seenLayouts[hash]
		if !exists {
			rows = append(rows, layout.AsRow())
			seenLayouts[hash] = struct{}{}
		}
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Date", "Title", "Notes", "Author"})
	table.AppendBulk(rows)
	table.Render()
}
