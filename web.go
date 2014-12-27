package main

import (
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/gorilla/mux"
	"github.com/hoisie/mustache"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

func languages(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	log.Print(params)

	var content []byte
	content, _ = ioutil.ReadFile("tasks.json")
	tasks := make([]map[string]string, 0)
	if err := json.Unmarshal(content, &tasks); err != nil {
		log.Print(err)
		http.Error(w, "internal json parsing error", 505)
		return
	}

	headerBytes, _ := ioutil.ReadFile("header.html")
	header := string(headerBytes)

	context := Context{Lang1: params["lang1"], Lang2: params["lang2"], Tasks: tasks}
	html := mustache.RenderFile("tasks.html", context)

	fmt.Fprintf(w, header+"\n"+html)
}

func codeblocks(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	log.Print(params)
	taskName := params["taskName"]

	resp, err := http.Get("http://rosettacode.org/wiki/" + taskName)
	if err != nil {
		http.Error(w, "couldn't open rosetta code", 505)
		return
	}

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		http.Error(w, "couldn't parse rosetta code", 503)
		return
	}

	lang1 := strings.ToLower(params["lang1"])
	lang2 := strings.ToLower(params["lang2"])
	code := map[int]string{1: "", 2: ""}
	matching := 0

	doc.Find("#mw-content-text h2, pre").EachWithBreak(func(i int, s *goquery.Selection) bool {
		if s.Is("h2") {
			lang := strings.ToLower(strings.Trim(s.Find("span.mw-headline").Text(), " "))
			if lang == lang1 {
				matching = 1
			} else if lang == lang2 {
				matching = 2
			} else if len(code[1]) > 0 && len(code[2]) > 0 {
				return false
			} else {
				matching = 0
			}
		} else if matching != 0 {
			html, err := s.Html()
			if err != nil {
				return true
			}
			code[matching] = code[matching] + "<pre>" + html + "</pre>"
		}
		return true
	})

	if len(code[1]) == 0 || len(code[2]) == 0 {
		http.Error(w, "code not found for these two languages", 404)
		return
	}

	context := Context{Lang1: code[1], Lang2: code[2]}
	html := mustache.RenderFile("codeblock.html", context)
	fmt.Fprintf(w, html)
}

func redirectToSlash(w http.ResponseWriter, req *http.Request) {
	http.Redirect(w, req, req.URL.String()+"/", 301)
	return
}

type Context struct {
	Lang1 string
	Lang2 string
	Tasks []map[string]string
}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/compare/{lang1}/{lang2}/", languages)
	router.HandleFunc("/compare/{lang1}/{lang2}", redirectToSlash)
	router.HandleFunc("/codeblock/{lang1}/{lang2}/{taskName}/", codeblocks)
	router.HandleFunc("/codeblock/{lang1}/{lang2}/{taskName}", redirectToSlash)
	http.Handle("/", router)

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}
	log.Print("listening...")
	http.ListenAndServe(":"+port, nil)
}
