package service

import (
	_ "embed"
	"html/template"
	"strings"
)

//go:embed static/html/base.html
var BASEHTML string

//go:embed static/html/index.html
var INDEXHTML string

//go:embed static/html/admin.html
var ADMINHTML string

//go:embed static/html/user.html
var USERHTML string

//go:embed static/html/userReg.html
var USERREGHTML string

//go:embed static/html/login.html
var LOGINHTML string
