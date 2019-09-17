package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type room struct {
	// forward is a channel that holds incoming messages
	// that should be forwarded to the other clients
	forward chan []byte
	// join is a channel for clients wishing to join the room
	join chan *client
	// leave is a channel for clients wishing to leave the room
	leave chan *client
	// clients hold all current clients in this room
	clients map[*client]bool
}

// newRoom makes a new room and returns it
func newRoom() *room {
	return &room{
		forward: make(chan []byte),
		join:    make(chan *client),
		leave:   make(chan *client),
		clients: make(map[*client]bool),
	}
}

// this function runs indefinitely
// until the program is terminated
func (r *room) run() {
	for {
		select {
		case client := <-r.join:
			// joining the chat
			r.clients[client] = true
		case client := <-r.leave:
			// leaving the chat
			delete(r.clients, client)
			close(client.send)
		case msg := <-r.forward:
			// forward message to all clients
			for client := range r.clients {
				client.send <- msg
			}
		}
	}
}

// turn the room into HTTP handler
// setup for sockets
const (
	socketBufferSize  = 1024
	messageBufferSize = 256
)

// the connection has to be upgraded because otherwise,
// the app would have to be accessed via a web socket
// rather than a web browser
var upgrader = &websocket.Upgrader{ReadBufferSize: socketBufferSize, WriteBufferSize: socketBufferSize}

func (r *room) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	socket, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Fatal("ServeHTTP:", err)
		return
	}
	client := &client{
		socket: socket,
		send:   make(chan []byte, messageBufferSize),
		room:   r,
	}
	r.join <- client
	defer func() { r.leave <- client }()
	// spin up go routine for client.write
	// non-blocking
	go client.write()
	// blocking by desing
	// will end when the runtime ends
	client.read()
}
