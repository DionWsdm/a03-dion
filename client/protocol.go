package main

import (
	"a03"
	"a03/utils"
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"maps"
	"net"
	"strconv"
	"strings"
	"time"
)

// Grader contract: retain these aliases and all exported function signatures.
type HttpRequest = a03.HttpRequest
type HttpResponse = a03.HttpResponse

func getTotalHeadersLength(headers a03.Headers) int {
	totalLen := 0
	keys := maps.Keys(headers)

	for key := range keys {
		totalLen += len(key) + 4 + len(headers[key])
	}

	return totalLen + 2
}

func getHeadersBytes(headers a03.Headers) []byte {
	totalLen := getTotalHeadersLength(headers)
	headersBytes := make([]byte, totalLen)
	keys := maps.Keys(headers)

	index := 0

	for key := range keys {
		line := key + ": " + headers[key] + "\r\n"
		index += copy(headersBytes[index:], []byte(line))
	}

	copy(headersBytes[index:], []byte("\r\n"))
	// fmt.Println("headersBytes:", headersBytes)
	return headersBytes
}

func getRequestError(requestLine string, headers a03.Headers) error {
	var err error

	if !utils.IsHeadersValid(headers, "client") {
		// fmt.Println("Headers invalid")
		err = a03.ErrBadRequest
	} else if !utils.IsRequestLineValid(requestLine) {

		// fmt.Println("Request line invalid:", requestLine)
		err = a03.ErrBadRequest
	}

	return err
}

// RequestEncoder builds the request line, headers, and terminating empty line.
// Encode the supplied headers and always send Connection: close, replacing any
// supplied value. Omit Accept-Encoding unless the caller set it (gzip). Return an
// error, never panic, for a missing/empty Host or Accept, CR or LF in a name or
// value, a colon or whitespace in a header name, whitespace in the URI, an empty
// or non-token Method, or a Version other than empty or HTTP/1.1. Always write
// HTTP/1.1 on the request line.
func RequestEncoder(request HttpRequest) ([]byte, error) {
	// TODO A: encode a request with CRLF line endings, without net/http.
	method := request.Method
	uri := request.URI
	version := request.Version
	headers := request.Headers

	requestLine := method + " " + uri + " " + version + "\r\n"

	err := getRequestError(requestLine, headers)
	if err != nil {
		return nil, err
	}

	// request line dibuat ulangn karena perintah "Always write HTTP/1.1 on the request line."
	requestLine = method + " " + uri + " " + "HTTP/1.1" + "\r\n"
	n := len(requestLine)

	// fmt.Println("accept-encoding:", headers["accept-encoding"])
	if !strings.Contains(headers["accept-encoding"], "gzip") {
		delete(headers, "accept-encoding")
	}

	lenRequestBytes := n + getTotalHeadersLength(headers) + 2
	requestBytes := make([]byte, lenRequestBytes)

	copy(requestBytes[0:], []byte(requestLine))
	copy(requestBytes[n:], getHeadersBytes(headers))

	return requestBytes, err
}

func decompressGzip(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return decompressed, err
}

func splitResponseStr(message string) (string, string, string, string) {
	var index int
	index = strings.Index(message, " ")
	version := message[0:index]
	// fmt.Println("version:", version)

	message = message[index+1:]
	index = strings.Index(message, " ")
	statusCodeStr := message[0:index]
	// fmt.Println("statusCodeStr:", statusCodeStr)

	message = message[index+1:]
	index = strings.Index(message, "\r\n")
	message = message[index+2:]

	index = strings.LastIndex(message, "\r\n\r\n")
	headersStr := message[0:index]
	// fmt.Println("headersStr:", headersStr)

	// fmt.Println("message[index+4]:", string(message[index+4:]))
	body := message[index+4:]
	// fmt.Println("body:", body)

	return version, statusCodeStr, headersStr, body
}

func getHeadersMap(headersStr string) (map[string]string, error) {
	headers := make(map[string]string)

	reader := bufio.NewReader(strings.NewReader(headersStr))
	line, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error 1")
		return nil, err
	}

	for len(line) > 2 {
		keyValue := strings.SplitN(line, ": ", 2)

		if len(keyValue) == 2 {
			headers[keyValue[0]] = strings.TrimSuffix(keyValue[1], "\r\n")
		} else {
			return nil, fmt.Errorf("invalid header line: %q", line)
		}

		line, err = reader.ReadString('\n')
		if err != nil && err != io.EOF {
			fmt.Println("err here")
			return nil, err
		}
	}

	return headers, nil
}

// ResponseDecoder validates a complete HTTP/1.1 response and its Content-Length.
// Body must contain exactly the declared number of wire bytes before optional
// gzip decompression. Keep Content-Length unchanged in the decoded headers.
func ResponseDecoder(message []byte) (HttpResponse, error) {
	// TODO B: parse the status line and case-insensitive headers, validate framing,
	// and decompress gzip. Reject truncated headers/bodies and invalid lengths.
	var err error
	messageStr := string(message)
	// fmt.Println("messageStr:", messageStr)
	version, statusCodeStr, headersStr, body := splitResponseStr(messageStr)

	statusCode, err := strconv.Atoi(statusCodeStr)
	if err != nil {
		fmt.Println(err.Error())
		return HttpResponse{}, err
	}

	headers, err := getHeadersMap(headersStr)
	if err != nil {
		fmt.Println(err.Error())
		return HttpResponse{}, err
	}

	bodyBytes := []byte(body)
	// fmt.Println("bodyBytes:", bodyBytes)

	contentLength, err := strconv.Atoi(strings.TrimSpace(headers["Content-Length"]))
	if err != nil {
		fmt.Println(err.Error())
		return HttpResponse{}, err
	}

	if contentLength < 0 || len(bodyBytes) != contentLength {
		return HttpResponse{}, fmt.Errorf("invalid Content-Length")
	}

	if strings.EqualFold(strings.TrimSpace(headers["content-encoding"]), "gzip") {
		bodyBytes, err = decompressGzip(bodyBytes)
		if err != nil {
			fmt.Println(err.Error())
			return HttpResponse{}, err
		}
	}

	response := HttpResponse{
		Version:    version,
		StatusCode: statusCode,
		Headers:    headers,
		Body:       bodyBytes,
	}

	return response, err
}

// Fetch sends one request and reads one complete response. It must use
// RequestEncoder and ResponseDecoder. The caller owns and closes conn.
func Fetch(request HttpRequest, conn net.Conn) (HttpResponse, error) {
	// TODO C: set a finite deadline, write all request bytes, read through CRLF CRLF,
	// then read exactly Content-Length body bytes. A Read is not a message boundary.
	// A buffered reader may already contain body bytes: do not discard them.
	err := conn.SetDeadline(time.Now().Add(30 * time.Second))
	defer func() {
		fmt.Println("Closing connection...")
		err = conn.Close()
		fmt.Println("Connection closed")
	}()

	fmt.Println("Encoding request...")
	requestEncoded, err := RequestEncoder(request)
	if err != nil {
		fmt.Println(err.Error())
		return HttpResponse{}, err
	}

	fmt.Println("Sending request to server...")
	_, err = conn.Write(requestEncoded)
	if err != nil {
		fmt.Println(err.Error())
		return HttpResponse{}, err
	}

	fmt.Println("Waiting response from server...")
	reader := bufio.NewReader(conn)

	var response []byte

	contentLength := 0

	for !strings.HasSuffix(string(response), "\r\n\r\n") {
		line, err := reader.ReadString('\n')
		// fmt.Println("line:", line)
		if err != nil {
			return HttpResponse{}, err
		}

		if strings.Contains(line, "Content-Length") {
			trimmed := strings.Trim(line, "\r\n")
			index := strings.Index(line, ": ")
			contentLength, err = strconv.Atoi(trimmed[index+2:])
			if err != nil {
				fmt.Println(err.Error())
				return HttpResponse{}, err
			}
		}
		response = append(response, []byte(line)...)
		// fmt.Println("Will stop:", strings.HasSuffix(string(response), "\r\n\r\n"))
	}

	body := make([]byte, contentLength)
	_, err = io.ReadFull(reader, body)
	if err != nil {
		fmt.Println(err.Error())
		return HttpResponse{}, err
	}
	response = append(response, body...)

	fmt.Println("Decoding response...")
	// fmt.Println("response:", response)
	decodedResponse, err := ResponseDecoder(response)

	fmt.Println("\n----RESPONSE----")
	fmt.Println("Version:", decodedResponse.Version)
	fmt.Println("Status Code:", decodedResponse.StatusCode)
	fmt.Println("Headers:", decodedResponse.Headers)
	fmt.Println("Body:", string(decodedResponse.Body))

	return decodedResponse, err
}
