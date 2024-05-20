package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// App Helpers
func handleError(msg string, err error) {
	fmt.Printf("Encountered error:\n%s\n%v", msg, err)
	os.Exit(1)
}

const DEBUG = false

func debug(msg string) {
	if DEBUG == true {
		fmt.Println(msg)
	}
}

// http Helpers

func routePatternIsFound(pattern string) bool {
	_, ok := routes[pattern]
	if ok {
		debug(fmt.Sprintf("Route pattern found: %s", pattern))
		return true
	} else {
		debug(fmt.Sprintf("Route not found: %s", pattern))
		return false
	}
}
func tryRouteHandler(pattern string, value string) (Http_Response, error) {
	debug(fmt.Sprintln("Trying handler for route: ", pattern))
	if handler, exists := routes[pattern]; exists {
		debug("Found route handler, executing...")
		response := handler(value)
		return response, nil
	} else {
		debug("Could not find handler!")
		return Http_Response{}, fmt.Errorf("No handler found for route pattern: %v", pattern)
	}
}

func checkRoutePatterns(method string, target string) Http_Response {
	tpath := strings.Clone(target)
	debug("Checking Route Patterns in 'exactness' priority order...\n")

	segs := strings.Split(tpath, "/")
	segCount := len(segs)
	lastIndex := strings.LastIndex(tpath, "/")
	lastPath := ""
	if lastIndex != -1 && lastIndex < len(tpath)-1 {
		lastPath = tpath[lastIndex+1:]
	}
	debug(fmt.Sprintf("Last Index: ", lastIndex))
	debug(fmt.Sprintf("Last Path: ", lastPath))
	debug(fmt.Sprintf("Segments: ", segCount))

	routeFound := false
	for i, seg := range segs {
		placeHolder := strings.Repeat("/{str}", i+1)
		debug(fmt.Sprintf("Current seg: %s Current i: %d Current vtest: %s\r\n", seg, i, placeHolder))
		if i == 0 {
			//exact match only, no vars
			//searchPaths[fmt.Sprintf("%s %s", method, tpath)] = ""
			searchPath := fmt.Sprintf("%s %s", method, tpath)
			routeFound = routePatternIsFound(searchPath)
			if routeFound {
				debug(fmt.Sprintln("exact path found: ", searchPath))
				response, err := tryRouteHandler(searchPath, "")
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
		if routeFound != true && idx != -1 && idx < len(tpath) {
			leftPath := tpath[:idx]
			value := target[idx+1:]
			nextSearch := fmt.Sprintf("%s %s%s", method, leftPath, placeHolder)
			debug(fmt.Sprintf("\r\nChecking route: %s with value: %s\r\n", nextSearch, value))
			routeFound = routePatternIsFound(nextSearch)
			if routeFound {
				debug(fmt.Sprintln("alternate path found: ", nextSearch))
				response, err := tryRouteHandler(nextSearch, value)
				if err != nil {
					handleError("Error in trying to execute handler", err)
				}
				return response
			}
			tpath = leftPath
			//searchPaths[nextSearch] = value
		}

	}
	// No patterns found
	if routeFound != true {
		debug("No matching route patterns found!")
	}
	return NOTFOUND
}

func stringByteLenAsString(s string) string {
	debug("Calculating string content length in bytes...")
	length := len([]byte(s))
	valueStr := strconv.Itoa(length)
	debug(fmt.Sprintf("Calculated Bytes: %s", valueStr))
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

var NOTFOUND = Http_Response{
	Version: HTTPV,
	Status:  404,
	Reason:  "Not Found",
	Headers: map[string]string{"Content-Type": "text/plain"},
	Body:    "",
}

// Request Handlers

/*
func matchRequestRoute(req Http_Request, reqMap map[string]string) (string, map[string]string) {
	target := req.Target
  matchRoutes :=
	//TODO: look for exact path match
	//TODO: if no exact path match, peel off end and assign to $1, and repear search for /path/$1

	//pathSegements := strings.Split(target, "/")

}
*/

func handleRequests(conn net.Conn, req Http_Request) {
	debug("Handling a new connection request...")
	debug("Building route search map...")
	response := checkRoutePatterns(req.Method, req.Target)
	responseWriter(conn, response)
}

// Response Handlers

func responseWriter(conn net.Conn, resp Http_Response) {
	debug("Sending connection response...")
	response := buildResponseString(resp)
	debug("String returned from buildResponseString()")
	debug("---------")
	debug(response)
	debug("---------")
	writer := bufio.NewWriter(conn)
	writeResult, err := writer.WriteString(response)
	debug(fmt.Sprintf("writeResult: %v", writeResult))
	if err != nil {
		handleError("Unable to write response", err)
	}
	writer.Flush()
	debug(fmt.Sprintf("Sent response: %s", response))
}

// Route Handlers
// var routes = make(map[string]Http_Response)
// var routes = make(map[string]func() Http_Response)
var routes = make(map[string]func(string) Http_Response)

func rootHandler(val string) Http_Response {
	resp := Http_Response{
		Version: HTTPV,
		Status:  200,
		Reason:  "OK",
		Headers: map[string]string{"Content-Type": "text/plain"},
		Body:    CRLF,
	}
	return resp
}

func echoHandler(val string) Http_Response {
	resp := Http_Response{
		Version: HTTPV,
		Status:  200,
		Reason:  "OK",
		Headers: map[string]string{"Content-Type": "text/plain"},
		Body:    val,
	}
	return resp
}

func define_routes() {
	debug("Routes being defined...")

	routes["GET /"] = rootHandler
	routes["GET /echo/{str}"] = echoHandler

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
			debug(fmt.Sprintf("Parsed method: %s\nParsed target: %s\nParsed version: %s", method, target, version))
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

	debug(fmt.Sprintf("Headers:\r\n%s", headers))
	debug(fmt.Sprintf("Body:\r\n%s", string(body)))
	debug(fmt.Sprintf("Received request:\r\n%s", requestString))

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

	define_routes()

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
		handleConnection(conn)
	}

}
