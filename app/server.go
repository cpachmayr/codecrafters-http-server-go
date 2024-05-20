package main

import (
	//"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"time"

	//"io"
	//"net"
	"os"

	//"regexp"
	//"strconv"
	//"strings"

	"github.com/gorilla/mux"
)

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

/*
	type Http_Response struct {
		Version string
		Status  int
		Reason  string
		Headers map[string]string
		Body    string
	}

	type Http_Request struct {
		Method  string
		Target  string
		Version string
		Headers map[string]string
		Body    string
	}

	type RouteHandler struct {
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

// var routes = make(map[string]Http_Response)
var routes = make(map[*regexp.Regexp]func(Http_Response) Http_Response)

	var NOTFOUND = Http_Response{
		Version: HTTPV,
		Status:  404,
		Reason:  "Not Found",
		Headers: map[string]string{"Content-Type": "text/plain"},
		Body:    "",
	}

	func define_routes() {
		debug("Routes being defined...")

		routes["GET /"] = Http_Response{
			Version: HTTPV,
			Status:  200,
			Reason:  "OK",
			Headers: map[string]string{"Content-Type": "text/plain"},
			Body:    CRLF,
		}

		debug("Routes ready.")
	}

	func handle_requests(conn net.Conn, req Http_Request) {
		debug("Handling a new connection request...")
		match := req.Method + " " + req.Target
		resp, ok := routes[match]
		if ok {
			debug(fmt.Sprintf("Route found: %s", match))
			debug(fmt.Sprintf("Response:\n%s", buildResponseString(resp)))
		} else {
			debug(fmt.Sprintf("Route not found: %s", match))
			debug(fmt.Sprintf("Response:\n%s", buildResponseString(NOTFOUND)))
			resp = NOTFOUND
		}
		sendResponse(conn, resp)
	}

	func buildHeaderString(headers map[string]string) string {
		headerString := ""
		if len(headers) == 0 {
			return CRLF
		}
		for key, value := range headers {
			headerString += fmt.Sprintf("%s: %s\r\n", key, value)
		}
		return headerString
	}

	func mapRequestHeaders(headers string) map[string]string {
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

	func getContentLengthHeaderString(s string) string {
		debug("Calculating Body content length in bytes...")
		length := len([]byte(s))
		valueStr := strconv.Itoa(length)
		debug(fmt.Sprintf("Calculated Bytes: %s", valueStr))
		return fmt.Sprintf("Content-Length: %s\r\n", valueStr)
	}

	func buildResponseString(resp Http_Response) string {
		debug("building response string from Http_Response fields...")
		debug("---------")
		headers := buildHeaderString(resp.Headers)
		debug("Adding Content-Length to headers...")
		headers += getContentLengthHeaderString(resp.Body)
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

	func sendResponse(conn net.Conn, resp Http_Response) {
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

	func handleConnection(conn net.Conn) {
		debug("Handling connection...")
		defer conn.Close()
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
				return
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
			return
		}

		// Build Http_Request type
		connRequest := Http_Request{
			Method:  method,
			Target:  target,
			Version: version,
			Headers: mapRequestHeaders(headers),
			Body:    string(body),
		}

		debug(fmt.Sprintf("Headers:\r\n%s", headers))
		debug(fmt.Sprintf("Body:\r\n%s", string(body)))
		debug(fmt.Sprintf("Received request:\r\n%s", requestString))

		handle_requests(conn, connRequest)

}
*/
func EchoHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	//w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, vars["str"])
}

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")

	var wait time.Duration

	r := mux.NewRouter()
	// r.HandleFunc("/", HomeHandler)
	r.HandleFunc("/echo/{str}", EchoHandler)
	http.Handle("/", r)

	srv := &http.Server{
		Addr:         "0.0.0.0:4221",
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	<-c

	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()

	srv.Shutdown(ctx)

	log.Println("shutting down")
	os.Exit(0)
	/*
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

	*/
}
