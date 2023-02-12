package go_web_page_parser

import (
	"github.com/team-ide/go-web-page-parser/downloader"
	"testing"
)

func TestDownloader(t *testing.T) {
	var url = `https://gbasedbt.com/dl/docs/%e5%ae%89%e8%a3%85%e8%bf%90%e7%bb%b4%e4%bc%98%e5%8c%96/`
	var dir = `C:/Workspaces/Code/teamide/go-web-page-parser/temp`
	var err error

	err = downloader.Parser(url, dir, nil)

	if err != nil {
		panic(err)
	}
}
