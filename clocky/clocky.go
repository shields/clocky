package clocky

import (
	"http"
	"template"
	"time"
)

type data struct {
	BigTime, SmallTime string
}

var tmpl = template.Must(template.New("page").Parse(PAGE))

func handler(w http.ResponseWriter, r *http.Request) {
	now := time.LocalTime()
	d := data{
		BigTime:   now.Format("3:04"),
		SmallTime: now.Format(":05&thinsp;pm"),
	}
	tmpl.Execute(w, d)
}

func init() {
	http.HandleFunc("/", handler)
}

const PAGE = `<!DOCTYPE html>
<head>
	<title>Clocky</title>
</head>
<div>{{.BigTime}}{{.SmallTime}}</div>`
