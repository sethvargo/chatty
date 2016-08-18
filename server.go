package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	sillyname "github.com/Pallinder/sillyname-go"
	"github.com/tylerb/graceful"

	redis "gopkg.in/redis.v4"

	"golang.org/x/net/websocket"
)

const (
	channelName string = "chatty"

	typeMessage  string = "message"
	typeAnnounce string = "announce"
)

type Server struct {
	// redis is the client for communicating with redis
	redis *redis.Client

	// clients is the mapping of client user names to websocket connections this
	// instance is responsible for.
	clients     map[string]*websocket.Conn
	clientsLock sync.RWMutex

	httpServer *graceful.Server

	stopCh chan struct{}
}

func NewServer() (*Server, error) {
	client := redis.NewClient(&redis.Options{
		Addr: ":6379",
	})

	// Verify connection
	_, err := client.Ping().Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %s", err)
	}

	server := &Server{
		redis:   client,
		clients: make(map[string]*websocket.Conn),
		stopCh:  make(chan struct{}, 1),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		c, err := Asset("public/index.html")
		if err != nil {
			panic(err)
		}
		w.Write(c)
	})
	mux.HandleFunc("/_healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	mux.Handle("/ws", websocket.Handler(server.wsHandler))

	server.httpServer = &graceful.Server{
		Timeout: 5 * time.Second,
		Server: &http.Server{
			Addr:    ":8080",
			Handler: mux,
		},
	}

	return server, nil
}

func (s *Server) wsHandler(ws *websocket.Conn) {
	username := s.generateRandomUsername(ws)
	hash := fmt.Sprintf("%x", md5.Sum([]byte(username)))

	defer func() {
		s.Leave(&LeaveOpts{
			Username: username,
		})
	}()

	s.Join(&JoinOpts{
		Username: username,
		Conn:     ws,
	})

	for {
		var contents []byte
		err := websocket.Message.Receive(ws, &contents)
		if err != nil && err != io.EOF {
			log.Printf("[ERR] ws error: %s", err)
			break
		}

		trim := strings.TrimSpace(string(contents))
		if trim == "" {
			time.Sleep(250 * time.Millisecond)
			continue
		}

		s.Publish(&PublishOpts{
			Type:     typeMessage,
			Username: username,
			MD5:      hash,
			Message:  trim,
		})
	}
}

func (s *Server) Start() error {
	// Start the server
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil {
			log.Printf("[ERR] server: %s", err)
		}
	}()

	// Start the pub-sub
	pubsub, err := s.redis.Subscribe(channelName)
	if err != nil {
		return fmt.Errorf("error subscribing: %s", err)
	}

	errCh := make(chan error)
	msgCh := make(chan *redis.Message, 1)
	go func() {
		for {
			m, err := pubsub.ReceiveMessage()
			if err != nil {
				log.Printf("[ERR] pubsub receive: %s", err)
				errCh <- err
				return
			}

			select {
			case msgCh <- m:
			case <-s.stopCh:
				return
			}
		}
	}()

	for {
		select {
		case m := <-msgCh:
			func() {
				s.clientsLock.RLock()
				defer s.clientsLock.RUnlock()
				for _, conn := range s.clients {
					conn.Write([]byte(m.Payload))
				}
			}()
		case err := <-errCh:
			return err
		case <-s.stopCh:
			return nil
		}
	}
}

func (s *Server) Stop() {
	s.clientsLock.RLock()
	defer s.clientsLock.RUnlock()
	for _, conn := range s.clients {
		defer conn.Close()
	}

	close(s.stopCh)
}

type JoinOpts struct {
	Username string
	Conn     *websocket.Conn
}

func (s *Server) Join(i *JoinOpts) {
	s.clientsLock.Lock()
	defer s.clientsLock.Unlock()

	s.clients[i.Username] = i.Conn

	s.Publish(&PublishOpts{
		Type:    typeAnnounce,
		Message: fmt.Sprintf("%s joined!", i.Username),
	})
}

type LeaveOpts struct {
	Username string
}

func (s *Server) Leave(i *LeaveOpts) {
	s.clientsLock.Lock()
	defer s.clientsLock.Unlock()

	if conn, ok := s.clients[i.Username]; ok {
		defer conn.Close()
		delete(s.clients, i.Username)
		s.Publish(&PublishOpts{
			Type:    typeAnnounce,
			Message: fmt.Sprintf("%s left!", i.Username),
		})
	}
}

type PublishOpts struct {
	Type     string `json:"type"`
	Username string `json:"username"`
	MD5      string `json:"md5"`
	Message  string `json:"message"`
}

func (s *Server) Publish(i *PublishOpts) error {
	c, err := json.Marshal(i)
	if err != nil {
		return err
	}

	_, err = s.redis.Publish(channelName, string(c)).Result()
	if err != nil {
		return fmt.Errorf("error publishing message: %s", err)
	}
	return nil
}

func (s *Server) generateRandomUsername(conn *websocket.Conn) string {
	s.clientsLock.Lock()
	defer s.clientsLock.Unlock()

	var name string
	for {
		name = sillyname.GenerateStupidName()
		if _, ok := s.clients[name]; !ok {
			s.clients[name] = conn
			break
		}
	}

	return name
}
