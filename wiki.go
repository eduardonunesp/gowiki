package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var funcMap = template.FuncMap{
	"setLink": setLink,
}

var templates = template.Must(template.New("main").Funcs(funcMap).ParseGlob(filepath.Join("tmpl", "*.html")))
var validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")
var wikiLink = regexp.MustCompile("\\[([a-zA-Z0-9]+)\\]")

type Page struct {
	Title   string
	Body    []byte
	IsFront bool
}

func NewPage(title string) *Page {
	return &Page{Title: title}
}

func NewPageWithBody(title string, body []byte) *Page {
	return &Page{Title: title, Body: body}
}

func (p *Page) save() error {
	filename := "data/" + p.Title + ".txt"
	return ioutil.WriteFile(filename, p.Body, 0600)
}

func setLink(body []byte) template.HTML {
	replacer := func(s []byte) []byte {
		_, s = s[len(s)-1], s[:len(s)-1]
		_, s = s[0], s[1:]
		link := fmt.Sprintf("<a href=\"/view/%s\">%s</a>", string(s), string(s))
		return []byte(link)
	}

	return template.HTML(wikiLink.ReplaceAllFunc(body, replacer))
}

func loadPage(title string) (*Page, error) {
	filename := "data/" + title + ".txt"
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return NewPageWithBody(title, body), nil
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	if strings.ToLower(p.Title) == "frontpage" || p.Title == "" {
		p.IsFront = true
	}

	err := templates.ExecuteTemplate(w, tmpl+".html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}

	renderTemplate(w, "view", p)
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		p = NewPage(title)
	}
	renderTemplate(w, "edit", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	body := r.FormValue("body")
	p := NewPageWithBody(title, []byte(body))
	err := p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, m[2])
	}
}

func frontPageHandler(w http.ResponseWriter, r *http.Request) {
	title := "FrontPage"
	p, err := loadPage(title)
	if err != nil {
		p = NewPage(title)
		renderTemplate(w, "edit", p)
	} else {
		renderTemplate(w, "view", p)
	}
}

func main() {
	if _, err := os.Stat("data"); os.IsNotExist(err) {
		err := os.Mkdir("data", 0755)
		if err != nil {
			log.Fatal(err)
		}
	}

	http.HandleFunc("/", frontPageHandler)
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))

	http.ListenAndServe("localhost:8000", nil)
}
