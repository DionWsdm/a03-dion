package utils

import (
	"a03"
	"fmt"
	"maps"
	"regexp"
	"strings"
)

func IsHeaderLineValid(headerLine string) bool {
	isValid := true

	if !strings.Contains(headerLine, "\r\n") {
		isValid = false
	} else if !strings.Contains(headerLine, ": ") {
		isValid = false
	}

	return isValid
}

func IsHeadersValid(headers a03.Headers, side string) bool {
	isValid := true

	host := headers["host"]
	accept := headers["accept"]

	if len(headers) > 8192 || host == "" {
		fmt.Println("Headers len invalid")
		isValid = false
	} else if side == "client" && accept == "" {
		fmt.Println("Headers len invalid")
		isValid = false
	} else {
		keys := maps.Keys(headers)

		for key := range keys {
			if strings.Contains(key, "\r") || strings.Contains(key, "\n") {
				fmt.Println("Header contains \\r or \\n")
				isValid = false
			} else {
				value := headers[key]
				if strings.Contains(value, "\r") || strings.Contains(value, "\n") {
					fmt.Println("Header value contains \\r or \\n")
					isValid = false
				}
			}
		}
	}

	return isValid
}

func IsRequestLineValid(requestLine string) bool {
	isValid := true

	if !strings.Contains(requestLine, "\r\n") {
		isValid = false
	}

	requestLine = strings.TrimSuffix(requestLine, "\r\n")

	pattern := regexp.MustCompile(`^[^ \r\n]+ [^ \r\n]* HTTP/1\.1$`)
	if !pattern.MatchString(requestLine) {
		isValid = false
	}

	return isValid
}
