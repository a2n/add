package main

import (
	"fmt"
	"flag"
	"os"
	"net/http"
	"errors"
	"strings"
	"io/ioutil"
	"encoding/json"
	"time"
	"strconv"
	"code.google.com/p/go.net/html"
)

func checkStatus(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

type InitService struct {
	OriginalPath string
	WorkingPath string
}

type Document struct {
	Title string
	Id string
	Latest int64
	Current int64
	URL string
	PdfURL string
}

type Library struct {
	Documents []interface{} `json:"documents"`
}

func NewInitService() *InitService {
	return &InitService{
		OriginalPath: "",
	}
}

func (s *InitService) Init() {
	initOpt := flag.String("init", "", "init PATH, set the working folder to PATH.")

	flag.Parse()

	// Get original path
	orig, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	s.OriginalPath = orig

	// Making working folder
	if strings.Index(*initOpt, "/") == 0 {
		// Absolute path
		s.WorkingPath = *initOpt
	} else {
		// Relative path
		s.WorkingPath = fmt.Sprintf("%s/%s", s.OriginalPath, *initOpt)
		fmt.Println(s.WorkingPath)
	}

	err = os.Mkdir(s.WorkingPath, os.ModePerm)
	if err != nil {
		if os.IsExist(err) {
			fmt.Println(err)
			os.Exit(-1)
		} else {
			s.OnError(err)
		}
	}

	fmt.Printf("Initlizing a working folder in %s", s.WorkingPath)

	err = os.Chdir(s.WorkingPath)
	checkStatus(err)

	// Making metadata folder
	err = os.Mkdir(".add", os.ModePerm)
	checkStatus(err)

	// Move to metadata folder
	err = os.Chdir(".add")
	checkStatus(err)

	// Download library json
	const libraryURL = "https://developer.apple.com/library/ios/navigation/library.json"
	resp, err := http.DefaultClient.Get(libraryURL)
	checkStatus(err)
	if resp.StatusCode != 200 {
		s.OnError(errors.New("Fail to get library."))
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	docs, err := s.parseJson(data)
	if err != nil {
		panic(err)
	}
	fmt.Printf("count: %d\n", len(docs))
}

func (s *InitService) parseJson(data []byte) ([]Document, error) {
	/*
	data, err := ioutil.ReadFile("json")
	if err != nil {
		return nil, err
	}
	*/

	library := Library{}
	err := json.Unmarshal(data, &library)
	if err != nil {
		return nil, err
	}

	documents := make([]Document, 0)
	count := len(library.Documents)
	/*
	 * FIXME
	 * The server supports http 1.1 protocol, menas it should making a 
	 * persistent connection, and download content for speeding up.
	 *
	 */
	for k, v := range library.Documents {
		fmt.Printf("%d / %d\n", k, count)
		switch vv := v.(type) {
			case []interface{}:
				docType := vv[2].(float64)
				if docType != 3 && docType != 10 {
					// Only accept guide and reference.
					continue
				}

				slice := strings.Split(vv[3].(string), "-")

				year, err := strconv.Atoi(slice[0])
				if err != nil {
					panic(err)
				}

				month, err := strconv.Atoi(slice[1])
				if err != nil {
					panic(err)
				}

				day, err := strconv.Atoi(slice[2])
				if err != nil {
					panic(err)
				}

				date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC) 
				if err != nil {
					panic(err)
				}

				url := fmt.Sprintf("https://developer.apple.com/library/ios/navigation/%s", vv[9].(string))
				doc := Document {
					Title: vv[0].(string),
					Id: vv[1].(string),
					Latest: date.Unix(),
					Current: 0,
					URL: url,
					PdfURL: s.getPdfUrl(url),
				}
				documents = append(documents, doc)
		}
	}
	data, err = json.MarshalIndent(documents, "", "  ")
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile("json2", data, 0600)
	if err != nil {
		panic(err)
	}

	return documents, nil
}

func (s *InitService) getPdfUrl(url string) (string) {
	resp, err := http.DefaultClient.Get(url)
	if err != nil {
		return ""
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return ""
	}

	result := ""
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "meta" {
			for _, v := range n.Attr {
				if v.Key == "contents" {
					// FIXME: Add base url as prefix.
					result = v.Val
					break;
				}
			}
		}
		for node := n.FirstChild; node != nil; node = node.NextSibling {
			f(node)
		}
	}
	f(doc)
	return result
}

func (s *InitService) download() {
	const libraryURL = "https://developer.apple.com/library/ios/navigation/library.json"
	resp, err := http.DefaultClient.Get(libraryURL)
	checkStatus(err)
	if resp.StatusCode != 200 {
		s.OnError(errors.New("Fail to get library."))
	}
}

func (s *InitService) OnError(err error) {
	if err == nil {
		return
	}

	fmt.Println(err)

	// Remove working path
	err = os.Remove(s.WorkingPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	// Return original path
	err = os.Chdir(s.OriginalPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	os.Exit(-1)
}

func cmdOptions() {
	init := NewInitService()
	init.Init()
}

func main() {
	cmdOptions()
}
