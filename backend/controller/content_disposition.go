package controller

import (
	"fmt"
	"net/url"
	"strings"
)

func contentDisposition(disposition, fileName string) string {
	fileName = strings.TrimSpace(fileName)
	if fileName == "" {
		fileName = "download"
	}
	return fmt.Sprintf("%s; filename*=UTF-8''%s", disposition, url.PathEscape(fileName))
}
