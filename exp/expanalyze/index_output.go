package main

import (
	"bytes"
	"io/ioutil"
	"path"
	"text/template"
)

func generateIndex() error {
	b := new(bytes.Buffer)
	expTemplate := template.Must(template.New(templateIndex).Parse(templateIndex))
	if err := expTemplate.Execute(b, expreport); err != nil {
		return err
	}

	err := ioutil.WriteFile(path.Join(exproot, reportFolder, "index.html"), b.Bytes(), 0644)
	if err != nil {
		return err
	}

	return nil
}
