package main

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
	"strings"
	"text/template"
)

func languages(res http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	t, _ := template.New("index").Parse(`
<!doctype html>
<meta charset=utf-8>
<title>side-by-side languages</title>
<link href=https://cdn.rawgit.com/picnicss/picnic/master/releases/v1.1.min.css rel=stylesheet>
<style>
pre {
  white-space: pre-wrap;
  font-size: 60%;
}
</style>
<script src=//ajax.googleapis.com/ajax/libs/jquery/2.1.3/jquery.min.js></script>
<script>
$(function () {
  $(".example").each(function () {
    var container = $(this)
    $.get("/example/" + container.data("example") + "/{{ .Lang1 }}/{{ .Lang2 }}/", function (html) {
      if (!html) return
      html = '<h3>' + container.data("example") + '</h3>' + html
      container.replaceWith(html)
    })
  })
})
</script>

<div class="example" data-example="Write_language_name_in_3D_ASCII"></div>
<div class="example" data-example="Write_to_Windows_event_log"></div>
<div class="example" data-example="Zeckendorf_number_representation"></div>
<div class="example" data-example="Zero_to_the_zero_power"></div>
<div class="example" data-example="Zhang-Suen_thinning_algorithm"></div>
    `)
	err := t.Execute(res, Context{Lang1: params["lang1"], Lang2: params["lang2"]})
	if err != nil {
		log.Print(err)
	}
}

func codeblocks(res http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	log.Print(params)
	example := params["example"]

	resp, err := http.Get("http://rosettacode.org/wiki/" + example)
	if err != nil {
		http.Error(res, "oops", 505)
	}

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		http.Error(res, "oops", 505)
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
		return
	}

	t, _ := template.New("codeblocks").Parse(`
<div class="row">
  <div class="half">
    {{ .Lang1 }}
  </div>
  <div class="half">
    {{ .Lang2 }}
  </div>
</div>
    `)
	err = t.Execute(res, Context{Lang1: code[1], Lang2: code[2]})
	if err != nil {
		log.Print(err)
	}
}

type Context struct {
	Lang1 string
	Lang2 string
}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/{lang1}/{lang2}/", languages)
	router.HandleFunc("/example/{example}/{lang1}/{lang2}/", codeblocks)
	http.Handle("/", router)

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}
	log.Print("listening...")
	http.ListenAndServe(":"+port, nil)
}
