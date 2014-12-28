package main

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/kennygrant/sanitize"
	"net/http"
	"strings"
)

func TasksForLanguages(langs map[int]string) (tasks []map[string]string, err error) {
	tasks = make([]map[string]string, 0)
	cache := map[string]map[string]string{}
	counts := map[string]int{}
	for i := 1; i <= 2; i++ {
		tasksForThisLanguage, err := TasksForLanguage(langs[i])
		if err != nil {
			return nil, err
		}
		for _, task := range tasksForThisLanguage {
			cache[task["Href"]] = task
			counts[task["Href"]]++
		}
	}
	for href, count := range counts {
		if count > 1 {
			tasks = append(tasks, cache[href])
		}
	}

	return tasks, nil
}

func TasksForLanguage(lang string) (tasks []map[string]string, err error) {
	resp, err := http.Get("http://rosettacode.org/wiki/" + lang)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return nil, err
	}

	tasks = make([]map[string]string, 0)
	doc.Find("#mw-pages a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			path := strings.Split(strings.Trim(href, "/"), "/")
			href = path[len(path)-1]
			tasks = append(tasks, map[string]string{"Href": href, "Name": s.Text()})
		}
	})

	return tasks, nil
}

func CodeblockForTask(taskName string, langs map[int]string) (code map[int]string, err error) {
	resp, err := http.Get("http://rosettacode.org/wiki/" + taskName)
	if err != nil {
		return
	}

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return
	}

	code = map[int]string{}
	matching := 0

	doc.Find("#mw-content-text h2, pre").EachWithBreak(func(i int, s *goquery.Selection) bool {
		if s.Is("h2") {
			langName := strings.ToLower(strings.Trim(s.Find("span.mw-headline").Text(), " "))
			if langName == langs[1] {
				matching = 1
			} else if langName == langs[2] {
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
			html = sanitize.HTML(html)
			code[matching] = code[matching] + "<pre><code class=\"language-" + langs[matching] + "\">" + html + "</code></pre>"
		}

		return true
	})

	return code, nil
}
