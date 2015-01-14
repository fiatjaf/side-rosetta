package main

import (
	"encoding/json"
	"github.com/bmizerany/mc"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"text/template"
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

	headerC := Context{Title: "Side-by-side programming languages comparisons"}
	headerT, err := template.ParseFiles("header.html")
	if err != nil {
		log.Print(err)
	}
	headerT.Execute(w, headerC)

	context := Context{Languages: languages}
	t, err := template.ParseFiles("index.html")
	if err != nil {
		log.Print(err)
	}
	t.Execute(w, context)
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

	// cache this, please
	w.Header().Set("Cache-control", "public; max-age=5184000")

	headerC := Context{Title: strings.Title(langs[1]) + " x " + strings.Title(langs[2]) + " side-by-side"}
	headerT, err := template.ParseFiles("header.html")
	if err != nil {
		log.Print(err)
	}
	headerT.Execute(w, headerC)

	context := Context{Lang1: params["lang1"], Lang2: params["lang2"], Tasks: tasks}
	t, err := template.ParseFiles("tasks.html")
	if err != nil {
		log.Print(err)
	}
	t.Execute(w, context)
}

func codeblocks(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	log.Print(params)
	taskName := params["taskName"]
	if taskGroup, ok := params["taskGroup"]; ok {
		taskName = taskGroup + "/" + taskName
	}

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
					code[i] += "<pre><code class=\"language-" + langs[i] + "\">" + html + "</code></pre>"
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

	// cache this, please
	w.Header().Set("Cache-control", "public; max-age=5184000")

	context := Context{Lang1: code[1], Lang2: code[2]}
	t, err := template.ParseFiles("codeblock.html")
	if err != nil {
		log.Print(err)
	}
	t.Execute(w, context)
}

func redirectToSlash(w http.ResponseWriter, req *http.Request) {
	http.Redirect(w, req, req.URL.String()+"/", 301)
	return
}

func redirectToHome(w http.ResponseWriter, req *http.Request) {
	http.Redirect(w, req, "/", 302)
	return
}

type Context struct {
	Title     string
	Lang1     string
	Lang2     string
	Tasks     []map[string]string
	Languages []string
}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/", index)
	router.HandleFunc("/compare/", redirectToHome)
	router.HandleFunc("/compare", redirectToHome)
	router.HandleFunc("/compare/{lang1}/{lang2}/", languages)
	router.HandleFunc("/compare/{lang1}/{lang2}", redirectToSlash)
	router.HandleFunc("/codeblock/{lang1}/{lang2}/{taskName}/", codeblocks)
	router.HandleFunc("/codeblock/{lang1}/{lang2}/{taskName}", redirectToSlash)
	router.HandleFunc("/codeblock/{lang1}/{lang2}/{taskGroup}/{taskName}/", codeblocks)
	router.HandleFunc("/codeblock/{lang1}/{lang2}/{taskGroup}/{taskName}", redirectToSlash)
	http.Handle("/", router)

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}
	log.Print("listening...")
	http.ListenAndServe(":"+port, nil)
}
