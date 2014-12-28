package main

import (
	"encoding/json"
	"fmt"
	"github.com/bmizerany/mc"
	"github.com/gorilla/mux"
	"github.com/hoisie/mustache"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

func index(w http.ResponseWriter, req *http.Request) {
	var content []byte
	content, _ = ioutil.ReadFile("languages.json")
	languages := make([]string, 0)
	if err := json.Unmarshal(content, &languages); err != nil {
		log.Print(err)
		http.Error(w, "internal json parsing error", 505)
		return
	}

	context := Context{Languages: languages}
	html := mustache.RenderFile("index.html", context)

	headerBytes, _ := ioutil.ReadFile("header.html")
	header := string(headerBytes)
	fmt.Fprintf(w, header+"\n"+html)
}

func languages(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	log.Print(params)

	// try to get the list of suitable tasks from the live rosetta code page
	langs := map[int]string{1: params["lang1"], 2: params["lang2"]}
	tasks, err := TasksForLanguages(langs)
	if err != nil {
		log.Print(err)
		http.Error(w, "these languages are probably fake", 406)
		return
	} else if len(tasks) == 0 {
		// if nothing was found, return all
		tasks := make([]map[string]string, 0)
		var content []byte
		content, _ = ioutil.ReadFile("tasks.json")
		if err := json.Unmarshal(content, &tasks); err != nil {
			log.Print(err)
			http.Error(w, "internal json parsing error", 505)
			return
		}
	}

	context := Context{Lang1: params["lang1"], Lang2: params["lang2"], Tasks: tasks}
	html := mustache.RenderFile("tasks.html", context)

	headerBytes, _ := ioutil.ReadFile("header.html")
	header := string(headerBytes)
	fmt.Fprintf(w, header+"\n"+html)
}

func codeblocks(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	log.Print(params)
	taskName := params["taskName"]

	lang1 := strings.ToLower(params["lang1"])
	lang2 := strings.ToLower(params["lang2"])
	langs := map[int]string{1: lang1, 2: lang2}
	code := map[int]string{1: "", 2: ""}

	// try to found the code in memcache
	memcache, err := mc.Dial("tcp", os.Getenv("MEMCACHEDCLOUD_SERVERS"))
	if err == nil {
		err = memcache.Auth(os.Getenv("MEMCACHEDCLOUD_USERNAME"), os.Getenv("MEMCACHEDCLOUD_PASSWORD"))
		if err == nil {
			for i := 1; i <= 2; i++ {
				html, _, _, err := memcache.Get(taskName + "::" + langs[i])
				if err == nil && html != "" {
					code[i] = "<pre><code class=\"language-" + langs[i] + "\">" + html + "</code></pre>"
				}
			}
		}
	}

	if len(code[1]) == 0 || len(code[2]) == 0 {
		// nothing found on cache, search the html
		code, err = CodeblockForTask(taskName, langs)
		if err != nil {
			http.Error(w, "couldn't parse rosetta code", 505)
			return
		}
	}

	if len(code[1]) == 0 || len(code[2]) == 0 {
		http.Error(w, "code not found for these two languages", 404)
		return
	}

	// save code for this task in memcached
	memcache.Set(taskName+"::"+langs[1], code[1], 0, 0, 1296000)
	memcache.Set(taskName+"::"+langs[2], code[2], 0, 0, 1296000)

	context := Context{Lang1: code[1], Lang2: code[2]}
	html := mustache.RenderFile("codeblock.html", context)
	fmt.Fprintf(w, html)
}

func redirectToSlash(w http.ResponseWriter, req *http.Request) {
	http.Redirect(w, req, req.URL.String()+"/", 301)
	return
}

type Context struct {
	Lang1     string
	Lang2     string
	Tasks     []map[string]string
	Languages []string
}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/", index)
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
