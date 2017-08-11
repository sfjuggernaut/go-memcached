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
	cmdSet    = "set"
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
)

// "flags" is 32bits to support memcached 1.2.1
func getCmdInfo(scanner *bufio.Scanner) (cmd string, key string, flags uint32, expTime int32, n int, cas uint64, err error) {
	scanner.Scan()
	line := scanner.Text()
	if err = scanner.Err(); err != nil {
		return
	}
	if len(line) == 0 {
		err = errors.New("no command provided")
		return
	}

	args := strings.Split(line, " ")
	cmd = args[0]

	switch cmd {
	case cmdCas:
		_, err = fmt.Sscanf(line, "%s%s%d%d%d%d", &cmd, &key, &flags, &expTime, &n, &cas)
	case cmdDelete, cmdGet, cmdGets:
		key = args[1]
	case cmdSet:
		_, err = fmt.Sscanf(line, "%s%s%d%d%d", &cmd, &key, &flags, &expTime, &n)
	}

	return
}

func getValue(scanner *bufio.Scanner, n int) (value string, err error) {
	scanner.Scan()
	value = scanner.Text()
	if err = scanner.Err(); err != nil {
		return
	}
	if n != len(value) {
		err = errors.New("incorrect length")
		return
	}
	return
}

// Loop waiting for new commands to be received until either the
// client closes the connection or we pass our deadline.
//
// Currently only supports the text protocol.
func (server *Server) handleConnection(conn net.Conn) {
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	defer conn.Close()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	var reply string

	for {
		scanner := bufio.NewScanner(reader)

		cmd, key, _, _, n, cas, err := getCmdInfo(scanner)
		if err == io.EOF {
			// client closed the connection
			log.Printf("handleConnection: client (%s) closed the connection\n", conn.RemoteAddr())
			break
		}
		if err, ok := err.(net.Error); ok && err.Timeout() {
			// reached our deadline
			// XXX this is a hard deadline, doesn't refresh with activity
			log.Println("handleConnection: reached dedline")
			break
		}
		if err != nil {
			log.Println("handleConnection: error reading cmd info:", err)
			continue
		}

		// XXX need to support multiple keys
		if len(key) > maxKeyLength {
			reply = "CLIENT_ERROR key is too long (max is 250 bytes)\r\n"
			writer.WriteString(reply)
			writer.Flush()
			continue
		}

		switch cmd {
		case cmdCas:
			_, entryCas, err := server.LRU.Get(key)
			if err == cache.ErrCacheMiss {
				reply = replyNotFound
			} else if err != nil {
				reply = replyNotStored
			} else if cas != entryCas {
				reply = replyExists
			} else {
				value, err := getValue(scanner, n)
				if err != nil {
					reply = replyNotStored
				} else {
					server.LRU.Add(key, value)
					reply = replyStored
				}
			}
			writer.WriteString(reply)
			writer.Flush()

		case cmdGet:
			value, _, err := server.LRU.Get(key)
			if err != nil {
				reply = replyEnd
			} else {
				// XXX hard-coding flags to 0 now
				reply = fmt.Sprintf("VALUE %s %d %d%s%s%s%s", key, 0, len(value), endOfLine, value, endOfLine, replyEnd)
			}
			writer.WriteString(reply)
			writer.Flush()

		case cmdGets:
			value, cas, err := server.LRU.Get(key)
			if err != nil {
				reply = replyEnd
			} else {
				// XXX hard-coding flags to 0 now
				reply = fmt.Sprintf("VALUE %s %d %d %d%s%s%s%s", key, 0, len(value), cas, endOfLine, value, endOfLine, replyEnd)
			}
			writer.WriteString(reply)
			writer.Flush()

		case cmdSet:
			value, err := getValue(scanner, n)
			if err != nil {
				reply = replyNotStored
			} else {
				server.LRU.Add(key, value)
				reply = replyStored
			}
			writer.WriteString(reply)
			writer.Flush()

		case cmdDelete:
			err := server.LRU.Delete(key)
			if err != nil {
				reply = replyNotFound
			} else {
				reply = replyDeleted
			}
			writer.WriteString(reply)
			writer.Flush()

		default:
			log.Println("handleConnection: unsupported cmd:", cmd)
			reply = replyError
			writer.WriteString(reply)
			writer.Flush()
		}
	}
}
