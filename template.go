package main

import (
	"html/template"
)

type Tmpl struct {
	templateIndex *template.Template
	templateLogin *template.Template
	templateReg   *template.Template
}

func NewTmpl() *Tmpl {
	tmplIdx := template.Must(template.ParseFiles("./index.html"))
	tmplLog := template.Must(template.ParseFiles("./login.html"))
	tmplReg := template.Must(template.ParseFiles("./reg.html"))
	return &Tmpl{
		templateIndex: tmplIdx,
		templateLogin: tmplLog,
		templateReg:   tmplReg,
	}
}
