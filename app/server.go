package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// App Helpers
var DEBUG bool
var DIRPATH string

func handleError(msg string, err error) {
	fmt.Printf("Encountered error:\n%s\n%v", msg, err)
	os.Exit(1)
}

func pathExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		debug("File or directory does not exist.")
		return false
	} else {
		debug("File or directory exists.")
		return true
	}
}

func define_flags() {
	// Get flags from command line
	debugOn := flag.Bool("debug-on", true, "turn debugging on")
	debugOff := flag.Bool("debug-off", false, "turn debugging off")
	directory := flag.String("directory", "", "directory location")
	flag.Parse()
	if *debugOn {
		DEBUG = true
	} else {
		DEBUG = false
	}
	if *debugOff {
		DEBUG = false
	}
	DIRPATH = *directory
	debugf("--directory: %s found: %v", DIRPATH, pathExists(DIRPATH))

}

func debug(msg string) {
	if DEBUG {
		fmt.Println(msg)
	}
}

func debugf(msgformat string, args ...interface{}) {
	formattedMsg := fmt.Sprintf(msgformat, args...)
	debug(formattedMsg)
}

// http Helpers

func routePatternIsFound(pattern string) bool {
	_, ok := routes[pattern]
	if ok {
		debugf("Route pattern found: %s", pattern)
		return true
	} else {
		debugf("Route not found: %s", pattern)
		return false
	}
}
func tryRouteHandler(pattern string, value string, conn net.Conn, req Http_Request) (Http_Response, error) {
	debugf("Trying handler for route: ", pattern)
	if handler, exists := routes[pattern]; exists {
		debug("Found route handler, executing...")
		response := handler(value, conn, req)
		return response, nil
	} else {
		debug("Could not find handler!")
		return Http_Response{}, fmt.Errorf("No handler found for route pattern: %v", pattern)
	}
}

func checkRoutePatterns(conn net.Conn, req Http_Request) Http_Response {
	method := req.Method
	target := req.Target
	tpath := strings.Clone(target)

	debug("Checking Route Patterns in 'exactness' priority order...\n")

	segs := strings.Split(tpath, "/")
	segCount := len(segs)
	lastIndex := strings.LastIndex(tpath, "/")
	lastPath := ""
	if lastIndex != -1 && lastIndex < len(tpath)-1 {
		lastPath = tpath[lastIndex+1:]
	}
	debugf("Last Index: %d", lastIndex)
	debugf("Last Path: %s", lastPath)
	debugf("Segments: %d", segCount)

	routeFound := false
	for i, seg := range segs {
		placeHolder := strings.Repeat("/{str}", i+1)
		debugf("Current seg: %s Current i: %d Current vtest: %s\r\n", seg, i, placeHolder)
		if i == 0 {
			searchPath := fmt.Sprintf("%s %s", method, tpath)
			routeFound = routePatternIsFound(searchPath)
			if routeFound {
				debugf("exact path found: %s", searchPath)
				response, err := tryRouteHandler(searchPath, "", conn, req)
				if err != nil {
					handleError("Error in trying to execute handler", err)
				}
				return response
			}
		}

		/* If no exact path is found, try alternates with {str} placeholders
		substitue path elements right to left and replace with {str} for each variation */

		debug("No exact path match found... checking alternate routes")
		idx := strings.LastIndex(tpath, "/")
		if !routeFound && idx != -1 && idx < len(tpath) {
			leftPath := tpath[:idx]
			value := target[idx+1:]
			nextSearch := fmt.Sprintf("%s %s%s", method, leftPath, placeHolder)
			debugf("\r\nChecking route: %s with value: %s\r\n", nextSearch, value)
			routeFound = routePatternIsFound(nextSearch)
			if routeFound {
				debugf("alternate path found: %s", nextSearch)
				response, err := tryRouteHandler(nextSearch, value, conn, req)
				if err != nil {
					handleError("Error in trying to execute handler", err)
				}
				return response
			}
			tpath = leftPath
		}

	}
	// No patterns found
	if !routeFound {
		debug("No matching route patterns found!")
	}
	return NOT_FOUND
}

func stringByteLenAsString(s string) string {
	debug("Calculating string content length in bytes...")
	length := len([]byte(s))
	valueStr := strconv.Itoa(length)
	debugf("Calculated Bytes: %s", valueStr)
	return fmt.Sprintf("Content-Length: %s\r\n", valueStr)
}

func headersMapToString(headers map[string]string) string {
	headerString := ""
	if len(headers) == 0 {
		return CRLF
	}
	for key, value := range headers {
		headerString += fmt.Sprintf("%s: %s\r\n", key, value)
	}
	return headerString
}

func headersStringToMap(headers string) map[string]string {
	headersMap := make(map[string]string)
	headerLines := strings.Split(headers, CRLF)
	for i := 0; i < len(headerLines); i++ {
		keyValuePair := strings.Split(headerLines[i], ": ")
		if len(keyValuePair) != 2 {
			headersMap[headerLines[i]] = ""
		} else {
			key := keyValuePair[0]
			value := keyValuePair[1]
			headersMap[key] = value
		}
	}
	return headersMap
}

func buildResponseString(resp Http_Response) string {
	debug("building response string from Http_Response fields...")
	debug("---------")
	headers := headersMapToString(resp.Headers)
	debug("Adding Content-Length to headers...")
	headers += stringByteLenAsString(resp.Body)
	debug("---------")
	debug("Built headers string:\r\n")
	debug(headers)
	debug("Creating full resposne string...")
	debug("---------")
	respString := fmt.Sprintf("%s %d %s\r\n%s\r\n%s", resp.Version, resp.Status, resp.Reason, headers, resp.Body)
	debug("Full response string:")
	debug("---------")
	debug(respString)
	debug("---------")
	return respString
}

// http types and definitions

type Http_Request struct {
	Method  string
	Target  string
	Version string
	Headers map[string]string
	Body    string
}

type Http_Response struct {
	Version string
	Status  int
	Reason  string
	Headers map[string]string
	Body    string
}

type Route_Handler struct {
	Match   *regexp.Regexp
	Handler func(Http_Request) Http_Response
}

const (
	CRLF       = "\r\n"
	HTTPV      = "HTTP/1.1"
	DoubleCRLF = CRLF + CRLF
)

var OK = Http_Response{
	Version: HTTPV,
	Status:  200,
	Reason:  "OK",
	Headers: make(map[string]string),
	Body:    "" + CRLF,
}

var NOT_FOUND = Http_Response{
	Version: HTTPV,
	Status:  404,
	Reason:  "Not Found",
	Headers: map[string]string{"Content-Type": "text/plain"},
	Body:    "",
}

var BAD_REQUEST = Http_Response{
	Version: HTTPV,
	Status:  400,
	Reason:  "Bad Request",
	Headers: map[string]string{"Content-Type": "text/plain"},
	Body:    "",
}

var SERVER_ERROR = Http_Response{
	Version: HTTPV,
	Status:  500,
	Reason:  "Internal Server Error",
	Headers: map[string]string{"Content-Type": "text/plain"},
	Body:    "",
}

// Request Handlers

func handleRequests(conn net.Conn, req Http_Request) {
	debug("Handling a new connection request...")
	debug("Building route search map...")
	res := checkRoutePatterns(conn, req)
	// Check for Response Content Type
	contentType := res.Headers["Content-Type"]
	switch contentType {
	case "application/octet-stream":
		{
			responseFileWriter(conn, res)
		}
	case "text/plain":
		{
			responseWriter(conn, res)
		}
	}
}

// Response Handlers
func responseFileWriter(conn net.Conn, resp Http_Response) {
	debugf("Using responseFileWriter for file: %s", resp.Body)
	filePath := resp.Body
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Fprintf(conn, "%s %d %s\r\n", resp.Version, 404, "Not Found\r\n")
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		fmt.Fprintf(conn, "%s %d %s", resp.Version, 500, "Internal Server Error\r\n")
		handleError("Internal server error reading file info.", err)
		return
	}
	fileSize := fileInfo.Size()
	contentLength := fmt.Sprintf("%d", fileSize)
	resp.Headers["Content-Length"] = contentLength

	writer := bufio.NewWriter(conn)
	initResponse := fmt.Sprintf("%s %d %s\r\n", resp.Version, resp.Status, resp.Reason)
	debug("initResponse:\r\n")
	debug(initResponse)
	headersResponse := headersMapToString(resp.Headers)
	debug("headersResponse:\r\n")
	debug(headersResponse)

	writer.WriteString(initResponse)
	writer.WriteString(headersResponse)
	writer.WriteString(CRLF) // end of headers

	_, err = bufio.NewReader(file).WriteTo(writer)
	if err != nil {
		fmt.Fprintf(conn, "%s %d %s", resp.Version, 500, "Internal Server Error\r\n")
		handleError("Internal server error writing file.", err)
		return
	}
	writer.Flush()
}

func responseWriter(conn net.Conn, resp Http_Response) {
	debug("Sending connection response...")
	response := buildResponseString(resp)
	debug("String returned from buildResponseString()")
	debug("---------")
	debug(response)
	debug("---------")
	writer := bufio.NewWriter(conn)
	writeResult, err := writer.WriteString(response)
	debugf("writeResult: %v", writeResult)
	if err != nil {
		handleError("Unable to write response", err)
	}
	writer.Flush()
	debugf("Sent response: %s", response)
}

// Route Handlers

var routes = make(map[string]func(string, net.Conn, Http_Request) Http_Response)

func rootHandler(pathVals string, conn net.Conn, req Http_Request) Http_Response {
	resp := Http_Response{
		Version: HTTPV,
		Status:  200,
		Reason:  "OK",
		Headers: map[string]string{"Content-Type": "text/plain"},
		Body:    CRLF,
	}
	return resp
}

func echoHandler(pathVals string, conn net.Conn, req Http_Request) Http_Response {
	resp := Http_Response{
		Version: HTTPV,
		Status:  200,
		Reason:  "OK",
		Headers: map[string]string{"Content-Type": "text/plain"},
		Body:    pathVals,
	}
	return resp
}

func userAgentHandler(pathVals string, conn net.Conn, req Http_Request) Http_Response {
	agent := req.Headers["User-Agent"]
	resp := Http_Response{
		Version: HTTPV,
		Status:  200,
		Reason:  "OK",
		Headers: map[string]string{"Content-Type": "text/plain"},
		Body:    agent,
	}
	return resp
}

func fileRequestHandler(pathVals string, conn net.Conn, req Http_Request) Http_Response {
	// Set initial values, presume not found
	status := 404
	reason := "File not found."
	filename := pathVals

	dirPathExists := pathExists(DIRPATH)
	if !dirPathExists {
		debugf("Could not find dir path: %s", DIRPATH)
	}
	pathSep := string(os.PathSeparator)
	debugf("filename: %s", filename)
	fullpath := DIRPATH + pathSep + filename
	debugf("fullpath: %s", fullpath)
	filePathExists := pathExists(fullpath)
	if !filePathExists {
		debug("Path to file DOES NOT exist!")
	} else {
		debug("Path to file exists!")
		status = 200
		reason = "OK"
		file, err := os.Open(fullpath)
		if err != nil {
			handleError("File not found!", err)
			return NOT_FOUND
		}
		defer file.Close()
	}

	// set the body to fullpath, depend on writer to detect file for streaming to writer
	resp := Http_Response{
		Version: HTTPV,
		Status:  status,
		Reason:  reason,
		Headers: map[string]string{"Content-Type": "application/octet-stream", "Content-Disposition": "attachment; filename=" + filename},
		Body:    fullpath,
	}
	return resp
}

func filePostHandler(pathVals string, conn net.Conn, req Http_Request) Http_Response {
	debugf("filePostHandler request with vals: %s", pathVals)

	// Look for Content-Length
	contentLength := req.Headers["Content-Length"]
	length, err := strconv.Atoi(contentLength)
	if err != nil {
		return BAD_REQUEST
	}

	// Open file path for writing
	filePath := DIRPATH + string(os.PathSeparator) + pathVals
	debugf("Attempting to open filepath: %s", filePath)
	destFile, err := os.Create(filePath)
	if err != nil {
		return SERVER_ERROR
	}
	defer destFile.Close()

	// Create buffer reader for content
	reader := bufio.NewReader(conn)
	// Begin writing to file
	debug("Begin writing file...")
	bufWriter := bufio.NewWriter(destFile)
	buf := make([]byte, 1024)
	remaining := int64(length)
	for remaining > 0 {
		readSize := 1024
		if remaining < int64(readSize) {
			readSize = int(remaining)
		}
		n, err := reader.Read(buf[:readSize])
		if err != nil {
			debug("Error reading file!")
			return SERVER_ERROR
		}
		_, err = bufWriter.Write(buf[:n])
		if err != nil {
			debug("Error writing file!")
			return SERVER_ERROR
		}
		remaining -= int64(n)

		// Validate file writing
		debug("Validating and flushing...")
		err = bufWriter.Flush()
		if err != nil {
			debug("Problem flushing writer!")
			return SERVER_ERROR
		}
	}
	// Send Response
	resp := Http_Response{
		Version: HTTPV,
		Status:  201,
		Reason:  "OK",
		Headers: make(map[string]string),
		Body:    "" + CRLF,
	}
	return resp
}

func define_routes() {
	debug("Routes being defined...")

	routes["GET /"] = rootHandler
	routes["GET /echo/{str}"] = echoHandler
	routes["GET /user-agent"] = userAgentHandler
	routes["GET /files/{str}"] = fileRequestHandler
	routes["POST /files/{str}"] = filePostHandler

	debug("Routes ready.")
}

func connStringToRequest(conn net.Conn) Http_Request {

	requestString := ""
	reader := bufio.NewReader(conn)
	debug("bufio reader set, attempt to read all...")
	// Read lines
	lineCount := 0
	lines, headers := "", ""
	method, target, version := "", "", ""

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			handleError("Error reading lines: ", err)
			return Http_Request{}
		}
		if lineCount == 0 {
			// first line has the method, target and http version
			parts := strings.Split(line, " ")
			if len(parts) != 3 {
				err := fmt.Errorf("Line does not have three parts. Received: %v", line)
				handleError("First line doesn't appear to be a valid http request", err)
			}
			method = parts[0]
			target = parts[1]
			version = parts[2]
			debugf("Parsed method: %s\nParsed target: %s\nParsed version: %s", method, target, version)
		}
		if lineCount > 0 {
			// get headers
			headers += line
		}
		lineCount++

		if line == CRLF {
			break // end of headers
		}
		lines += line
	}

	// Get body content length
	contentLength := 0
	for _, line := range strings.Split(headers, CRLF) {
		if strings.HasPrefix(line, "Content-Length:") {
			contentLength, _ = strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:")))
			break
		}
	}

	// Read full content length of body
	body := make([]byte, contentLength)
	_, err := io.ReadFull(reader, body)
	if err != nil {
		handleError("Error reading body:", err)
		return Http_Request{}
	}

	// Build Http_Request type
	connRequest := Http_Request{
		Method:  method,
		Target:  target,
		Version: version,
		Headers: headersStringToMap(headers),
		Body:    string(body),
	}

	debugf("Headers:\r\n%s", headers)
	debugf("Body:\r\n%s", string(body))
	debugf("Received request:\r\n%s", requestString)

	return connRequest
}

func handleConnection(conn net.Conn) {
	debug("Handling connection...")
	defer conn.Close()
	connRequest := connStringToRequest(conn)
	handleRequests(conn, connRequest)
}

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")

	define_flags()
	define_routes()
	if DEBUG {
		fmt.Println("Debugging turned on")
	}
	listener, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		handleError("Failed to bind to 0.0.0.0 (localhost) port 4221", err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			handleError("Error accepting connection.", err)
			continue
		}
		go handleConnection(conn)
	}

}
