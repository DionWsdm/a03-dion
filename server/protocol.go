package main

import (
	"a03"
	"a03/utils"
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// Grader contract: retain these aliases and all exported function signatures.
type HttpRequest = a03.HttpRequest
type HttpResponse = a03.HttpResponse
type Student = a03.Student
type GreetResponse = a03.GreetResponse

// ServerStudent is configured before serving. Tests may replace it between cases.
// Do not change it while handling a connection.
var ServerStudent Student

func splitRequestStr(message string) (HttpRequest, error) {
	reader := bufio.NewReader(strings.NewReader(message))

	requestLine, err := reader.ReadString('\n')
	if err != nil {
		return HttpRequest{}, err
	}

	if !utils.IsRequestLineValid(requestLine) {
		fmt.Println("Request line invalid")
		return HttpRequest{}, a03.ErrBadRequest
	}

	headers := make(map[string]string)

	headerLine, err := reader.ReadString('\n')
	if err != nil {
		return HttpRequest{}, err
	}

	for len(headerLine) > 2 {

		if !utils.IsHeaderLineValid(headerLine) {
			fmt.Println("Header line invalid")
			return HttpRequest{}, a03.ErrBadRequest
		}

		keyValue := strings.Split(headerLine, ": ")
		headers[keyValue[0]] = strings.TrimSuffix(keyValue[1], "\r\n")

		headerLine, err = reader.ReadString('\n')
		if err != nil {
			return HttpRequest{}, err
		}
	}

	if !utils.IsHeadersValid(headers, "client") {
		fmt.Println("Headers invalid")
		return HttpRequest{}, a03.ErrBadRequest
	}

	requestLineSplitted := strings.Split(requestLine, " ")

	request := HttpRequest{
		Method:  requestLineSplitted[0],
		URI:     requestLineSplitted[1],
		Version: requestLineSplitted[2],
		Headers: headers,
	}
	return request, nil
}

// RequestDecoder validates a complete request header (including its terminator).
// Normalize header names to lowercase. Split header lines at the FIRST colon.
// Validate syntax and a nonempty Host before considering method or routing.
// Return an error for a header over 8192 bytes, a missing terminator, non-CRLF
// line endings, a request line that is not "<method> <origin-form URI> HTTP/1.1",
// a header line without a colon, or a missing/empty Host.
func RequestDecoder(message []byte) (HttpRequest, error) {
	// TODO D: validate HTTP/1.1 request syntax, CRLF, and the 8192-byte limit.
	// Do not reject syntactically valid methods here: HandleRequest produces 405.
	var err error

	if len(message) > 8192 {
		fmt.Println("Request header length invalid")
		return HttpRequest{}, a03.ErrBadRequest
	}

	messageStr := string(message)

	request, err := splitRequestStr(messageStr)
	if err != nil {
		fmt.Println(err.Error())
		return HttpRequest{}, err
	}

	return request, err
}

func isEndpoint(uri string, endpoint string) bool {
	isEndpoint := strings.HasPrefix(uri, endpoint)
	n := len(endpoint)

	if isEndpoint && len(uri) > n {
		if uri[n] != '?' {
			isEndpoint = false
		}
	}

	return isEndpoint
}

func getQAccept(acceptSplitted []string, acceptSearched string) (bool, float32) {
	n := len(acceptSplitted)

	found := false
	var q float32
	q = 0

	for i := 0; i < n; i++ {

		accept := acceptSplitted[i]
		if strings.Contains(accept, acceptSearched) {
			found = true
			qIndex := strings.Index(accept, "q=")
			if qIndex == -1 {
				q = 1
			} else {

				qValue := strings.TrimSpace(accept[qIndex+2:])
				qParsed, err := strconv.ParseFloat(qValue, 32)
				if err == nil {
					q = float32(qParsed)
				}
			}
		}
	}

	return found, q
}

// HandleRequest applies status precedence: syntax/Host (400), method (405),
// route/NPM (404), Accept (406), then success (200). It must also handle manually
// constructed requests from unit tests: an empty Host, a Version other than
// HTTP/1.1, or an empty or non-token (RFC 9110) Method is 400. Use ServerStudent
// as the identity. Route on the path before "?"; a query on / is ignored.
func HandleRequest(request HttpRequest) HttpResponse {
	// TODO E: GET / returns the exact UTF-8 identity message without a newline.
	// TODO F: validate /greet/<10-digit NPM>, compare NPM, and read optional name.
	// TODO G: serialize JSON/XML using simplified Accept/q negotiation; JSON wins
	// ties. Only successful /greet responses may select gzip from Accept-Encoding.
	// Error bodies are empty text/plain; 405 also has Allow: GET. Content-Type
	// values are exact, without parameters such as charset.
	// Placeholder only: A03 never returns 501.
	statusCode := 200
	responseHeaders := a03.Headers{"content-type": "text/plain", "connection": "close"}
	var body []byte
	var err error

	method := request.Method
	uri := request.URI
	version := request.Version
	headers := request.Headers

	requestLine := method + " " + uri + " " + version
	// fmt.Println("requestLine:", requestLine)
	// fmt.Println("headers:", headers)

	if !utils.IsRequestLineValid(requestLine) || !utils.IsHeadersValid(headers, "server") {
		fmt.Println("heree1")
		statusCode = 400
	} else if method != "GET" {
		statusCode = 405
	} else if isEndpoint(uri, "/greet/"+a03.NPM) {
		student := Student{
			Nama: a03.NAMA,
			Npm:  a03.NPM,
		}

		nameIndex := strings.Index(uri, "name=")
		andIndex := strings.Index(uri, "&")

		greeter := student.Nama
		if nameIndex != -1 && andIndex == -1 {
			greeter = uri[nameIndex+5:]
		} else if nameIndex != -1 {
			greeter = uri[nameIndex+5 : andIndex]
		}

		greetResponse := GreetResponse{
			Student: student,
			Greeter: greeter,
		}

		accept := headers["accept"]
		acceptSplitted := strings.Split(accept, ",")

		foundJson, qJson := getQAccept(acceptSplitted, "application/json")
		foundXml, qXml := getQAccept(acceptSplitted, "application/xml")

		if len(acceptSplitted) > 0 && !foundJson && !foundXml {
			statusCode = 406
		} else if qJson >= qXml {
			body, err = json.Marshal(greetResponse)
			if err != nil {
				fmt.Println(err.Error())
			}
		} else {
			body, err = xml.Marshal(greetResponse)
			if err != nil {
				fmt.Println(err.Error())
			}
		}
	} else if isEndpoint(uri, "/") {
		body = []byte("Halo, dunia! Aku " + a03.NAMA + " sedang mengerjakan A03")
	} else {
		statusCode = 404
	}

	return HttpResponse{
		Version:    a03.HTTPVersion,
		StatusCode: statusCode,
		Headers:    responseHeaders,
		Body:       body,
	}
}

// ResponseEncoder serializes a response using standard reason phrases and CRLF.
// Always compute Content-Length from the final wire bytes. Compress Body first
// when content-encoding is gzip and the status is 200; otherwise omit
// Content-Encoding. Always send Connection: close. Return an error for a status
// other than 200, 400, 404, 405, or 406, a missing Content-Type, or a Version
// other than empty or HTTP/1.1. Always write HTTP/1.1 on the status line.
func ResponseEncoder(response HttpResponse) ([]byte, error) {
	// TODO H: encode status line, required headers, empty line, and body.
	// Validasi status code
	reasonPhrases := map[int]string{
		200: "OK",
		400: "Bad Request",
		404: "Not Found",
		405: "Method Not Allowed",
		406: "Not Acceptable",
	}

	reason, exists := reasonPhrases[response.StatusCode]
	if !exists {
		return nil, fmt.Errorf("unsupported status code: %d", response.StatusCode)
	}

	// Validasi Content-Type
	contentType, exists := response.Headers["content-type"]
	if !exists || contentType == "" {
		return nil, fmt.Errorf("missing Content-Type")
	}

	// Validasi Version
	if response.Version != "" && response.Version != a03.HTTPVersion {
		return nil, fmt.Errorf("invalid HTTP version")
	}

	body := response.Body
	// fmt.Println("body here:", body)

	// Kompresi gzip hanya untuk response 200
	headers := response.Headers
	contentEncoding := headers["content-encoding"]

	if contentEncoding == "gzip" && response.StatusCode == 200 {
		fmt.Println("Zipping body...")
		var compressed bytes.Buffer

		writer := gzip.NewWriter(&compressed)

		_, err := writer.Write(body)
		if err != nil {
			fmt.Println(err.Error())
			return nil, err
		}

		err = writer.Close()
		if err != nil {
			fmt.Println(err.Error())
			return nil, err
		}

		body = compressed.Bytes()
		fmt.Println()
		contentEncoding = "gzip"
	} else {
		contentEncoding = ""
	}

	// Susun response headers
	var responseStr strings.Builder

	responseStr.WriteString(fmt.Sprintf(
		"HTTP/1.1 %d %s\r\n",
		response.StatusCode,
		reason,
	))

	responseStr.WriteString("Content-Type: " + contentType + "\r\n")
	responseStr.WriteString(fmt.Sprintf("Content-Length: %d\r\n", len(body)))
	responseStr.WriteString("Connection: close\r\n")

	if contentEncoding != "" {
		responseStr.WriteString("Content-Encoding: gzip\r\n")
	}

	responseStr.WriteString("\r\n")

	// Gabungkan headers dan body
	encoded := append([]byte(responseStr.String()), body...)

	// fmt.Println("encoded:", encoded)
	return encoded, nil
}

// HandleConnection serves exactly one request and one response, using the
// decoder, handler, and encoder above. The caller owns and closes conn.
func HandleConnection(conn net.Conn) error {
	// TODO I: set a finite deadline, accumulate headers up to 8192 bytes including
	// CRLF CRLF, decode, handle, encode, and write all response bytes.
	// Malformed/oversized input gets 400 if possible. A disconnected peer can be
	// closed without a response. Never panic or stop the outer accept loop.
	fmt.Println("Handling connection...")
	err := conn.SetDeadline(time.Now().Add(30 * time.Second))
	defer func() {
		fmt.Println("Closing connection...")
		conn.Close()
		fmt.Println("Connection closed")
	}()

	reader := bufio.NewReader(conn)
	var request []byte

	for !strings.HasSuffix(string(request), "\r\n\r\n") {
		line, err := reader.ReadString('\n')
		// fmt.Println("line:", line)
		if err != nil {
			return err
		}

		request = append(request, []byte(line)...)
		// fmt.Println("Will stop:", strings.HasSuffix(string(request), "\r\n\r\n"))
	}

	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	fmt.Println("Decoding request...")
	requestDecoded, err := RequestDecoder(request)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	fmt.Println("Handling request...")
	response := HandleRequest(requestDecoded)

	fmt.Println("Encoding response...")
	responseEncoded, err := ResponseEncoder(response)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	fmt.Println("Sending response to client...")
	_, err = conn.Write(responseEncoded)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	return err
}
