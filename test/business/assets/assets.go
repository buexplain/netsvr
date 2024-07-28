package assets

import (
	_ "embed"
	"netsvr/pkg/wd"
	"os"
	"path/filepath"
)

//go:embed client.html
var clientHtml string

func GetClientHtml() string {
	//优先读取本地文件，方便开发
	if b, _ := os.ReadFile(filepath.Join(wd.RootPath, "test", "business", "web", "client.html")); len(b) > 0 {
		return string(b)
	}
	return clientHtml
}
