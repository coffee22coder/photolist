package main

import (
	"html/template"
	textTemplate "text/template"
)

type Tmpl struct {
	templateIndex *textTemplate.Template
	templateLogin *template.Template
	templateReg   *template.Template
}

func NewTmpl() *Tmpl {
	tmplIdx := textTemplate.Must(textTemplate.ParseFiles("./index.html"))
	tmplLog := template.Must(template.ParseFiles("./login.html"))
	tmplReg := template.Must(template.ParseFiles("./reg.html"))
	return &Tmpl{
		templateIndex: tmplIdx,
		templateLogin: tmplLog,
		templateReg:   tmplReg,
	}
}
