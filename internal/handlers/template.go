package handlers

import (
	"html/template"
	"log"
	"net/http"
)

func RenderTemplate(t *template.Template, name string, data any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := t.ExecuteTemplate(w, name, data)

		if err != nil {
			log.Println(err)
		}
	}
}

func RenderTemplateFunc(t *template.Template, name string, dataFunc func(http.ResponseWriter, *http.Request) any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := t.ExecuteTemplate(w, name, dataFunc(w, r))

		if err != nil {
			log.Println(err)
		}
	}
}
