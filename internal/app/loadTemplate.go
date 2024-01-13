package service

import (
	_ "embed"
	"html/template"
	"strings"
)

func loadTemplate() (*template.Template, error) {
	// 加载模板

	//go:embed static/html/index.html
	var INDEXHTML string
	t, err := template.New("tIndex").Parse(INDEXHTML)
	if err != nil {
		return nil, err
	}

	//go:embed static/html/admin.html
	var ADMINHTML string
	t, err = t.New("tAdmin").Parse(ADMINHTML)
	if err != nil {
		return nil, err
	}

	//go:embed static/html/user.html
	var USERHTML string
	t, err = t.New("tUser").Parse(USERHTML)
	if err != nil {
		return nil, err
	}

	//go:embed static/html/userReg.html
	var USERREGHTML string
	t, err = t.New("tReg").Parse(USERREGHTML)
	if err != nil {
		return nil, err
	}

	//go:embed static/html/login.html
	var LOGINHTML string
	t, err = t.New("tLogin").Parse(LOGINHTML)
	if err != nil {
		return nil, err
	}

	t = t.Lookup("tIndex")
	return t, nil
}
