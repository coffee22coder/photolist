package main

import "html/template"

type Tmpl struct {
	template *template.Template
}

func NewTmpl() *Tmpl {
	tmpl := template.Must(template.ParseFiles("./index.html"))
	return &Tmpl{
		template: tmpl,
	}
}
