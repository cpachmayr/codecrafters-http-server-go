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
	"time"
)

// App Helpers
var DEBUGGER bool
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
	debugOn := flag.Bool("debug-on", false, "turn debugging on")
	debugOff := flag.Bool("debug-off", false, "turn debugging off")
	directory := flag.String("directory", "", "directory location")
	flag.Parse()
	if *debugOn {
		DEBUGGER = true
	} else {
		DEBUGGER = false
	}
	if *debugOff {
		DEBUGGER = false
	}
	DIRPATH = *directory
	// Add OS separator to the end if needed
	dirPathEnd := string(DIRPATH[len(DIRPATH)-1])
	pathSeparatorFound := dirPathEnd == string(os.PathSeparator)
	if !pathSeparatorFound {
		DIRPATH += string(os.PathSeparator)
	}
	debugf("--directory: %s found: %v", DIRPATH, pathExists(DIRPATH))

}

func debug(msg string) {
	if DEBUGGER {
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
	debugf("Trying handler for route: %s", pattern)
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

// Foo this is squat

func buildResponseString(res Http_Response) string {
	debug("building response string from Http_Response fields...")
	debug("---------")
	headers := headersMapToString(res.Headers)
	//debug("Adding Content-Length to headers...")
	//headers += stringByteLenAsString(res.Body)
	//debug("---------")
	debug("Built headers string:\r\n")
	debug(headers)
	debug("Creating full resposne string...")
	debug("---------")
	respString := fmt.Sprintf("%s %d %s\r\n%s\r\n%s", res.Version, res.Status, res.Reason, headers, res.Body)
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
	debug("Sending response to responseWriter")
	responseWriter(conn, res)
}

// Response Handlers
func fileResponseBody(dataPath string, res *Http_Response) (string, error) {
	debugf("Creating body from file: %s", dataPath)
	debugf("Calling Handler: %s", res.Headers["Request-Handler"])

	file, err := os.Open(dataPath)
	if err != nil {
		res.Status = 404
		res.Reason = "Not Found"
		return "Unable to open resource.", err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		res.Status = 500
		res.Reason = "Internal Server Error"
		return "Unable to get file information.", err
	}
	debugf("file.Stat() call returns: %v", fileInfo)

	fileSize := fileInfo.Size()
	debugf("fileInfo.Size() reports: %d", fileSize)
	contentLength := fmt.Sprintf("%d", fileSize)
	res.Headers["Content-Length"] = contentLength
	debugf("Content-Length header set to: %s", res.Headers["Content-Length"])

	//reader := bufio.NewReaderSize(file, 1024)
	body := make([]byte, fileSize)
	//	n, err := reader.Read(body)
	debug("Staring file.Read()")
	n, err := file.Read(body)
	if err != nil {
		res.Status = 500
		res.Reason = "Internal Server Error"
		return "Unable to read file.", err
	}
	if int64(n) != fileSize {
		res.Status = 500
		res.Reason = "Internal Server Error"
		return "Mismatch in file size and read bytes.", fmt.Errorf("Read %d bytes, expected %d", n, fileSize)
	}
	debugf("Expected %v bytes, Read %d bytes.", fileSize, n)
	return string(body), nil
}

func responseWriter(conn net.Conn, res Http_Response) {
	debug("Sending connection response...")
	response := buildResponseString(res)
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
	res := Http_Response{
		Version: HTTPV,
		Status:  200,
		Reason:  "OK",
		Headers: map[string]string{"Content-Type": "text/plain"},
		Body:    CRLF,
	}
	return res
}

func echoHandler(pathVals string, conn net.Conn, req Http_Request) Http_Response {
	res := Http_Response{
		Version: HTTPV,
		Status:  200,
		Reason:  "OK",
		Headers: map[string]string{"Content-Type": "text/plain"},
		Body:    pathVals,
	}
	return res
}

func userAgentHandler(pathVals string, conn net.Conn, req Http_Request) Http_Response {
	agent := req.Headers["User-Agent"]
	res := Http_Response{
		Version: HTTPV,
		Status:  200,
		Reason:  "OK",
		Headers: map[string]string{"Content-Type": "text/plain"},
		Body:    agent,
	}
	return res
}

func fileRequestHandler(pathVals string, conn net.Conn, req Http_Request) Http_Response {
	// Set initial res values, presume not found
	res := Http_Response{
		//TODO Set these values, so that filebody can work
		Version: HTTPV,
		Status:  404,
		Reason:  "Not Found",
		Headers: map[string]string{"Content-Type": "application/octet-stream"},
		Body:    "",
	}

	// Set filename from pathVals
	filename := pathVals
	debugf("filename: %s", filename)
	res.Headers["Content-Disposition"] = fmt.Sprintf("attachment; filename=%s", filename)

	// Set directory path
	dirPathExists := pathExists(DIRPATH)
	if !dirPathExists {
		debugf("Could not find dir path: %s", DIRPATH)
	}

	// Set Full Path
	fullpath := DIRPATH + filename
	debugf("fullpath: %s", fullpath)

	// Check for filepath and update response values
	filePathExists := pathExists(fullpath)
	if !filePathExists {
		debug("Path to file DOES NOT exist!")
		res.Status = 404
		res.Reason = "Not Found"
		res.Body = fmt.Sprintf("File path not found: %s", fullpath)
		return res
	}

	debug("Path to file exists!")
	res.Status = 200
	res.Reason = "OK"
	res.Headers["Request-Handler"] = "file-request-handler"

	debug("Attempting to load body from file...")
	body, err := fileResponseBody(fullpath, &res)
	if err != nil {
		handleError("File not found!", err)
		return res
	}
	res.Body = body

	return res
}

// return an http-style status int (e.g,. 201,400,500) status message string, and error status
func uploadHandler(fileLength int64, filename string, content string) (int, string, error) {
	var status int
	var reason string
	var errMsg error

	//TODO
	// Open file path for writing
	filePath := DIRPATH + filename
	debugf("Attempting to upload to filepath: %s", filePath)
	destFile, err := os.Create(filePath)
	if err != nil {
		status = 500
		reason = "Internal Server Error"
		errMsg = fmt.Errorf("Unable to create filepath: %s", filePath)
		return status, reason, errMsg
	}
	defer destFile.Close()

	// Commented out, trying to write waht is passed in via content
	// Create buffer reader for content
	debug("Reading request body content...")
	reader := strings.NewReader(content)

	// Begin writing to file
	debug("Establishing server writer...")
	bufWriter := bufio.NewWriter(destFile)
	buf := make([]byte, 1024)
	remaining := int64(fileLength)
	for remaining > 0 {
		readSize := 1024
		if remaining < int64(readSize) {
			readSize = int(remaining)
		}
		n, err := reader.Read(buf[:readSize])
		if err != nil {
			debug("Error reading file!")
			status = 500
			reason = "Internal Server Error"
			errMsg = fmt.Errorf("Unable to read from connection buffer.")
			return status, reason, errMsg
		}

		_, err = bufWriter.Write(buf[:n])
		if err != nil {
			debug("Error writing file!")
			status = 500
			reason = "Internal Server Error"
			errMsg = fmt.Errorf("Unable to write to file buffer.")
			return status, reason, errMsg
		}
		remaining -= int64(n)

		// Validate file writing
		debug("Validating and flushing...")
		err = bufWriter.Flush()
		if err != nil {
			debug("Problem flushing writer!")
			status = 500
			reason = "Internal Server Error"
			errMsg = fmt.Errorf("Unable to validate and flush file writer.")
			return status, reason, errMsg
		}
	}
	// Succesfully uploaded file.
	debug("Successfully uploaded file.")
	status = 201
	reason = "Created"
	errMsg = nil
	return status, reason, errMsg
}

func filePostHandler(pathVals string, conn net.Conn, req Http_Request) Http_Response {
	debugf("filePostHandler request with vals: %s", pathVals)
	var res Http_Response
	res.Version = HTTPV

	// Look for Content-Length
	contentLength := req.Headers["Content-Length"]
	length, err := strconv.Atoi(contentLength)
	if err != nil {
		res = BAD_REQUEST
		res.Headers["Error"] = "Content-Length header or length value missing."
		return BAD_REQUEST
	}
	debugf("Content-Length header or value missing. Received content length: %v", contentLength)

	status, reason, err := uploadHandler(int64(length), pathVals, req.Body)
	if err != nil {
		res = SERVER_ERROR
		res.Status = status
		res.Reason = reason
		res.Headers["Error"] = "Problem with uploading file."
		return res
	}

	// Successful file upload
	res = OK
	res.Status = status
	res.Reason = reason
	return res
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

		debugf("Found %d header lines: %s", lineCount, line)
		if line == CRLF {
			break // end of headers
		}
		lines += line
	}

	debug("End of headers, building Headers map...")
	headersMap := headersStringToMap(headers)
	debug("Headers map built, looking for Content-Type and Content-Length...")

	// Get body content length
	contentLength := 0
	contentType := ""
	contentLength, err := strconv.Atoi(headersMap["Content-Length"])
	if err != nil {
		contentLength = 0
	}
	debugf("contentLength: %d", contentLength)
	contentType = strings.TrimSpace(headersMap["Content-Type"])
	if len(contentType) == 0 {
		contentType = ""
	}
	debugf("contentType: %s", contentType)

	// Read full content length of body

	// Setting a 5-second read deadline to prevent blocking
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	debug("Attempting to get request body...")
	body := make([]byte, int64(contentLength))
	debugf("Ready to read %d bytes", contentLength)
	n, err := io.ReadFull(reader, body)
	debugf("Read %d bytes", n)
	if err != nil {
		if err == io.EOF {
			debug("Received EOF")
		} else if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			handleError("Timeout reading body: ", err)
		} else {
			handleError("Error reading body: ", err)
		}
		return Http_Request{}
	}

	debug("Building Http_Request")
	// Build Http_Request type
	connRequest := Http_Request{
		Method:  method,
		Target:  target,
		Version: version,
		Headers: headersStringToMap(headers),
		Body:    string(body),
	}

	return connRequest
}

func handleConnection(conn net.Conn) {
	debug("Handling connection...")
	defer conn.Close()
	connRequest := connStringToRequest(conn)
	go handleRequests(conn, connRequest)
}

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")

	define_flags()
	define_routes()
	if DEBUGGER {
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
