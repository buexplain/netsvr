package wd

import (
	"github.com/lesismal/nbio/logging"
	"os"
	"path/filepath"
	"strings"
)

// RootPath 程序根目录
var RootPath string

func init() {
	dir, err := os.Getwd()
	if err != nil {
		logging.Error("Get process working directory failed：%s", err)
		os.Exit(1)
	}
	RootPath = strings.TrimSuffix(filepath.ToSlash(dir), "/") + "/"
}
