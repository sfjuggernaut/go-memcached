package server

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

const (
	cmdGet    = "get"
	cmdGets   = "gets"
	cmdSet    = "set"
	cmdDelete = "delete"
)

// "flags" is 32bits to support memcached 1.2.1
func getCmdInfo(scanner *bufio.Scanner) (cmd, key string, flags uint32, expTime int32, n int, err error) {
	scanner.Scan()
	line := scanner.Text()
	if err = scanner.Err(); err != nil {
		return
	}

	_, err = fmt.Sscanf(line, "%s%s", &cmd, &key)
	if cmd == cmdSet {
		_, err = fmt.Sscanf(line, "%s%s%d%d%d", &cmd, &key, &flags, &expTime, &n)
	}

	return
}

func getSetInfo(scanner *bufio.Scanner, n int) (value string, err error) {
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
		cmd, key, _, _, n, err := getCmdInfo(scanner)
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
			break
		}

		if len(key) > maxKeyLength {
			reply = "CLIENT_ERROR key is too long (max is 250 bytes)\r\n"
			writer.WriteString(reply)
			writer.Flush()
			continue
		}

		switch cmd {
		case cmdGet, cmdGets:
			value, err := server.LRU.Get(key)
			if err != nil {
				reply = "END\r\n"
			} else {
				// XXX hard-coding flags to 0 now
				// XXX not returning cas yet
				reply = fmt.Sprintf("VALUE %s %d %d\r\n%s\r\nEND\r\n", key, 0, len(value), value)
			}
			writer.WriteString(reply)
			writer.Flush()

		case cmdSet:
			value, err := getSetInfo(scanner, n)
			if err != nil {
				reply = "NOT_STORED\r\n"
			} else {
				server.LRU.Add(key, value)
				reply = "STORED\r\n"
			}
			writer.WriteString(reply)
			writer.Flush()

		case cmdDelete:
			err := server.LRU.Delete(key)
			if err != nil {
				reply = "NOT_FOUND\r\n"
			} else {
				reply = "DELETED\r\n"
			}
			writer.WriteString(reply)
			writer.Flush()

		default:
			log.Println("handleConnection: unsupported cmd:", cmd)
			reply = "ERROR\r\n"
			writer.WriteString(reply)
			writer.Flush()
		}
	}
}
