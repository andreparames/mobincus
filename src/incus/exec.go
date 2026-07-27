package incus

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

func wsDialer() *websocket.Dialer {
	return &websocket.Dialer{
		NetDial: func(_, _ string) (net.Conn, error) {
			return net.DialTimeout("unix", DefaultUnixSocket, 10*time.Second)
		},
	}
}

func wsURL(opID, secret string) string {
	return fmt.Sprintf("ws://localhost/1.0/operations/%s/websocket?secret=%s", opID, secret)
}

func extractOpID(opURL string) string {
	for i := len(opURL) - 1; i >= 0; i-- {
		if opURL[i] == '/' {
			return opURL[i+1:]
		}
	}
	return opURL
}

func extractFDs(metadata json.RawMessage) map[string]string {
	if metadata == nil {
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(metadata, &m); err != nil {
		return nil
	}
	innerMeta, ok := m["metadata"]
	if !ok {
		return nil
	}
	innerMap, ok := innerMeta.(map[string]interface{})
	if !ok {
		return nil
	}
	fdsRaw, ok := innerMap["fds"]
	if !ok {
		return nil
	}
	fdsMap, ok := fdsRaw.(map[string]interface{})
	if !ok {
		return nil
	}
	fds := make(map[string]string, len(fdsMap))
	for k, v := range fdsMap {
		fds[k] = fmt.Sprintf("%v", v)
	}
	return fds
}

func extractReturn(metadata map[string]interface{}) int {
	if metadata == nil {
		return -1
	}
	retRaw, ok := metadata["return"]
	if !ok {
		return -1
	}
	retFloat, ok := retRaw.(float64)
	if !ok {
		return -1
	}
	return int(retFloat)
}

func (c *Client) ExecAndStream(name string, req ExecPost, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	resp, err := c.ExecInstance(name, req)
	if err != nil {
		return -1, err
	}
	if resp.Type != "async" {
		return -1, fmt.Errorf("expected async response, got %s", resp.Type)
	}

	opID := extractOpID(resp.Operation)
	fds := extractFDs(resp.Metadata)

	d := wsDialer()
	var wg sync.WaitGroup

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(sigCh)

	stream := func(fdKey string, writer io.Writer) {
		defer wg.Done()
		secret, ok := fds[fdKey]
		if !ok {
			return
		}
		conn, _, err := d.Dial(wsURL(opID, secret), nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if writer != nil {
				_, _ = writer.Write(msg)
			}
		}
	}

	wg.Add(2)
	go stream("1", stdout)
	go stream("2", stderr)

	if secret, ok := fds["0"]; ok {
		if stdin != nil {
			go func() {
				conn, _, err := d.Dial(wsURL(opID, secret), nil)
				if err != nil {
					return
				}
				defer conn.Close()
				buf := make([]byte, 32768)
				for {
					n, err := stdin.Read(buf)
					if n > 0 {
						conn.WriteMessage(websocket.BinaryMessage, buf[:n])
					}
					if err != nil {
						return
					}
				}
			}()
		} else {
			go func() {
				conn, _, _ := d.Dial(wsURL(opID, secret), nil)
				if conn != nil {
					select {}
				}
			}()
		}
	}

	if secret, ok := fds["control"]; ok {
		go func() {
			conn, _, err := d.Dial(wsURL(opID, secret), nil)
			if err != nil {
				return
			}
			defer conn.Close()

			readDone := make(chan struct{})
			go func() {
				for {
					_, _, err := conn.ReadMessage()
					if err != nil {
						close(readDone)
						return
					}
				}
			}()

			for {
				select {
				case sig := <-sigCh:
					msg, _ := json.Marshal(map[string]interface{}{
						"command": "signal",
						"signal":  int(sig.(syscall.Signal)),
					})
					conn.WriteMessage(websocket.TextMessage, msg)
				case <-readDone:
					return
				}
			}
		}()
	}

	op, err := c.WaitOperation(resp.Operation)
	wg.Wait()

	if err != nil {
		if strings.Contains(err.Error(), "Command not found") {
			return 127, nil
		}
		return -1, err
	}

	return extractReturn(op.Metadata), nil
}
