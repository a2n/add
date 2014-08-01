package main

import (
	"fmt"
	"flag"
	"os"
	"net/http"
	"errors"
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
	return &InitService{
		OriginalPath: "",
	}
}

func (s *InitService) Init() {
	initOpt := flag.String("init", "", "init PATH, set the working folder to PATH.")

	flag.Parse()

	orig, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	s.OriginalPath = orig

	s.WorkingPath = *initOpt
	err = os.Mkdir(s.WorkingPath, os.ModePerm)
	if err != nil && os.IsExist(err) {
		fmt.Println(err)
		os.Exit(-1)
	}

	fmt.Println("Created a working folder.")

	err = os.Chdir(s.WorkingPath)
	checkStatus(err)

	err = os.Mkdir(".add", os.ModePerm)
	checkStatus(err)

	libraryURL := "https://developer.apple.com/library/ios/navigation/library.jsona"
	resp, err := http.DefaultClient.Get(libraryURL)
	checkStatus(err)
	if resp.StatusCode != 200 {
		s.OnError(errors.New("Fail to get library."))
	}
}

func (s *InitService) Reset() (*error) {
	return nil
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
