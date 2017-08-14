package server

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"github.com/sfjuggernaut/go-memcached/pkg/cache"
)

const (
	cmdCas    = "cas"
	cmdDelete = "delete"
	cmdGet    = "get"
	cmdGets   = "gets"
	cmdQuit   = "quit"
	cmdSet    = "set"
	cmdHire   = "hireeric?"
)

const (
	endOfLine      = "\r\n"
	replyDeleted   = "DELETED\r\n"
	replyEnd       = "END\r\n"
	replyError     = "ERROR\r\n"
	replyExists    = "EXISTS\r\n"
	replyNotFound  = "NOT_FOUND\r\n"
	replyNotStored = "NOT_STORED\r\n"
	replyStored    = "STORED\r\n"
	replyYes       = "totes\r\n"
)

var (
	ErrInsufficientArgs = errors.New("Insufficient args")
)

// Request stores the information for a single client request
type Request struct {
	cmd  string
	keys []string
	// flags is 32bits to support memcached 1.2.1
	flags     uint32
	expTime   int32
	n         int
	cas       uint64
	dataBlock string
	err       error
}

// parseRequest verifies and parses the incoming request
func parseRequest(line string) (r Request, err error) {
	if len(line) == 0 {
		err = errors.New("no command provided")
		return
	}

	args := strings.Split(line, " ")
	r.cmd = args[0]

	switch r.cmd {
	case cmdCas:
		r.keys = make([]string, 1)
		_, err = fmt.Sscanf(line, "%s%s%d%d%d%d", &r.cmd, &r.keys[0], &r.flags, &r.expTime, &r.n, &r.cas)
	case cmdDelete:
		if len(args) < 2 {
			err = ErrInsufficientArgs
			return
		}
		r.keys = make([]string, 1)
		r.keys[0] = args[1]
	case cmdGet, cmdGets:
		r.keys = make([]string, len(args)-1)
		for i := 0; i < len(args)-1; i++ {
			r.keys[i] = args[i+1]
		}
	case cmdSet:
		r.keys = make([]string, 1)
		_, err = fmt.Sscanf(line, "%s%s%d%d%d", &r.cmd, &r.keys[0], &r.flags, &r.expTime, &r.n)
	}
	return
}

// continually consumes input from the connection
func connReader(scanner *bufio.Scanner, requests chan Request) {
	var line string

	for {
		// scan for cmd
		if valid := scanner.Scan(); !valid {
			// done scanning for this connection
			requests <- Request{err: io.EOF}
			break
		}
		line = scanner.Text()
		if err := scanner.Err(); err != nil {
			requests <- Request{err: err}
			continue
		}
		request, err := parseRequest(line)
		if err != nil {
			request.err = err
			requests <- request
			continue
		}

		// scan for data block if SET or CAS
		if request.cmd == cmdSet || request.cmd == cmdCas {
			// wait for data block
			if valid := scanner.Scan(); !valid {
				// done scanning for this connection
				requests <- Request{err: io.EOF}
				break
			}
			data := scanner.Text()
			if err := scanner.Err(); err != nil {
				requests <- Request{err: err}
				continue
			}
			if len(data) > request.n {
				requests <- Request{err: errors.New("data block provided is too long")}
				continue
			}
			request.dataBlock = data
		}
		requests <- request
	}
}

// Loop waiting for new commands to be received until either the
// client closes the connection, we pass our deadline, or receive
// quit signal.
//
// Currently only supports the text protocol.
func (server *Server) handleConnection(conn net.Conn) {
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	defer conn.Close()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	scanner := bufio.NewScanner(reader)
	var reply string

	requests := make(chan Request)
	go connReader(scanner, requests)

Loop:
	for {
		select {
		case request := <-requests:
			if request.err == io.EOF {
				// client closed the connection
				log.Printf("handleConnection: client (%s) closed the connection\n", conn.RemoteAddr())
				break Loop
			}
			if err, ok := request.err.(net.Error); ok && err.Timeout() {
				// reached our deadline
				// XXX this is a hard deadline, doesn't refresh with activity
				log.Println("handleConnection: reached dedline")
				break Loop
			}
			if request.err != nil {
				reply = fmt.Sprintf("CLIENT_ERROR %s%s", request.err, endOfLine)
				writer.WriteString(reply)
				writer.Flush()
				continue
			}

			if request.cmd == cmdQuit {
				// close connection for the client
				break Loop
			}

			// XXX need to support multiple keys
			for i := 0; i < len(request.keys); i++ {
				if len(request.keys[i]) > maxKeyLength {
					reply = fmt.Sprintf("CLIENT_ERROR key is too long (max is 250 bytes)%s", endOfLine)
					writer.WriteString(reply)
					writer.Flush()
					continue
				}
			}

			switch request.cmd {
			case cmdCas:
				_, _, entryCas, err := server.Cache.Get(request.keys[0])
				if err == cache.ErrCacheMiss {
					reply = replyNotFound
				} else if err != nil {
					reply = replyNotStored
				} else if request.cas != entryCas {
					reply = replyExists
				} else {
					server.Cache.Add(request.keys[0], request.dataBlock, request.flags)
					reply = replyStored
				}
				writer.WriteString(reply)
				writer.Flush()
				StatsNumCas.Add(1)

			case cmdDelete:
				err := server.Cache.Delete(request.keys[0])
				if err != nil {
					reply = replyNotFound
				} else {
					reply = replyDeleted
				}
				writer.WriteString(reply)
				writer.Flush()
				StatsNumDelete.Add(1)

			case cmdGet:
				for _, key := range request.keys {
					value, flags, _, err := server.Cache.Get(key)
					if err == nil {
						reply = fmt.Sprintf("VALUE %s %d %d%s%s%s", key, flags, len(value), endOfLine, value, endOfLine)
						writer.WriteString(reply)
					}
				}
				writer.WriteString(replyEnd)
				writer.Flush()
				StatsNumGet.Add(1)

			case cmdGets:
				for _, key := range request.keys {
					value, flags, cas, err := server.Cache.Get(key)
					if err == nil {
						reply = fmt.Sprintf("VALUE %s %d %d %d%s%s%s", key, flags, len(value), cas, endOfLine, value, endOfLine)
						writer.WriteString(reply)
					}
				}
				writer.WriteString(replyEnd)
				writer.Flush()
				StatsNumGets.Add(1)

			case cmdSet:
				server.Cache.Add(request.keys[0], request.dataBlock, request.flags)
				reply = replyStored
				writer.WriteString(reply)
				writer.Flush()
				StatsNumSet.Add(1)

			case cmdHire:
				writer.WriteString(replyYes)
				writer.Flush()

			default:
				log.Println("handleConnection: unsupported cmd:", request.cmd)
				reply = replyError
				writer.WriteString(reply)
				writer.Flush()
				StatsErrNumUnsupportedCmds.Add(1)
			}
		case <-server.quit:
			break Loop
		}
	}
}
