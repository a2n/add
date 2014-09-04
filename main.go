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

func NewInitService() *InitService {
    return &InitService{}
}

func (s *InitService) Init(target string) {
	// Get original path
    orig, err := os.Getwd()
    if err != nil {
        fmt.Println(err)
        os.Exit(-1)
    }
    s.OriginalPath = orig

    // Making working folder
    if strings.Index(target, "/") == 0 {
        // Absolute path
        s.WorkingPath = target
    } else {
        // Relative path
        s.WorkingPath = fmt.Sprintf("%s/%s", s.OriginalPath, target)
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

    fmt.Printf("Initlizing a working folder in %s\n", s.WorkingPath)

    err = os.Chdir(s.WorkingPath)
    checkStatus(err)

    // Making metadata folder
    err = os.Mkdir(".add", os.ModePerm)
    checkStatus(err)

    // Move to metadata folder
    err = os.Chdir(".add")
    checkStatus(err)

/*
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
	*/

	path := fmt.Sprintf("%s/library.json", s.OriginalPath)
	data, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}

    docs, err := s.parseJson(data)
    if err != nil {
        panic(err)
    }
    fmt.Printf("count: %d\n", len(docs))
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
    /*
     * FIXME
     * The server supports http 1.1 protocol, menas it should making a 
     * persistent connection, and download content for speeding up.
     *
     */
	const PREFIX = "https://developer.apple.com/library/ios"
	for k, v := range library.Documents {
		fmt.Printf("%d / %d\n", k, len(library.Documents))
        switch vv := v.(type) {
            case []interface{}:
                docType := vv[2].(float64)
                if docType != 3 {
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

                url := fmt.Sprintf("%s/%s", PREFIX, strings.Trim(vv[9].(string), "../"))
                pdfUrl := fmt.Sprintf("%s/%s", PREFIX, s.getPdfUrl(url))
                doc := Document {
                    Title: vv[0].(string),
                    Id: vv[1].(string),
                    Latest: date.Unix(),
                    Current: 0,
                    URL: url,
                    PdfURL: pdfUrl,
                }
                documents = append(documents, doc)
        }
    }

	/*
	// Get PDF URL
    count := len(documents)
	for k, v := range documents {
        fmt.Printf("%d / %d\n", k, count)
		pdfURL := fmt.Sprintf("%s/%s", PREFIX, s.getPdfUrl(v.URL))
		v.PdfURL = pdfURL
	}
	*/

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
    initOpt := flag.String("init", "", "init PATH, set the PATH as working folder.")
	flag.Parse()

    init := NewInitService()
	if len(*initOpt) > 0 {
		init.Init(*initOpt)
	}
}

type PersistentConnectionService struct {
}

func NewPersistentConectionService() *PersistentConnectionService {
	return &PersistentConnectionService{}
}

func (s *PersistentConnectionService) Dial() {
}

func (s *PersistentConnectionService) HangUp() {
}

func foo() {
	orig, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	path := fmt.Sprintf("%s/a/.add/json2", orig)
	data, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	err := json.Unmarshal(data, &library)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func main() {
	//cmdOptions()
	foo()
}
