package main

import (
	"fmt"
	"net"
	"os"
)

func handleError(msg string, err error) {
	fmt.Printf("Encountered error:\n%s\n%v", msg, err)
	os.Exit(1)
}

const (
	CRLF  = "\r\n"
	HTTPV = "HTTP/1.1"
)

type Http_Response struct {
	Version string
	Status  int
	Reason  string
	Headers string
	Body    string
}

func main() {
	type Http_Response struct {
		Version string
		Status  int
		Reason  string
		Headers string
		Body    string
	}
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		handleError("Failed to bind to port 4221", err)
	}

	conn, err := l.Accept()
	if err != nil {
		handleError("Error accepting connection", err)
	}

	respTest := Http_Response{
		Version: HTTPV,
		Status:  200,
		Reason:  "OK",
		Headers: CRLF,
		Body:    CRLF,
	}

	resp := fmt.Sprintf("%s %d %s\r\n%s\r\n%s", respTest.Version, respTest.Status, respTest.Reason, respTest.Headers, respTest.Body)

	_, err = conn.Write([]byte(resp))
	if err != nil {
		handleError("Problem writing response", err)
	}

	/*
		resp := Http_Response{
			Version: "HTTP/1.1",
			Status:  200,
			Reason:  "OK",
			Headers: CRLF,
			Body:    CRLF,
		}
	*/
	// r := fmt.Sprintf("%s %d %s%s", resp.Version, resp.Status, resp.Headers, resp.Body)
	// conn.Write([]byte("HTTP/1.1 200 OK\r\nHello World"))
	// fmt.Println("Response", r)
}
