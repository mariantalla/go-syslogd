package main

import (
	"fmt"
	"log"
	"net/http"

	syslog "gopkg.in/mcuadros/go-syslog.v2"
)

type RingBuffer struct {
	in  syslog.LogPartsChannel
	out syslog.LogPartsChannel
}

func NewRingBuffer(in syslog.LogPartsChannel, out syslog.LogPartsChannel) *RingBuffer {
	return &RingBuffer{in, out}
}

func (r *RingBuffer) Run() {
	for v := range r.in {
		select {
		case r.out <- v:
		default:
			<-r.out
			r.out <- v
		}
	}
	close(r.out)
}

func main() {
	in := make(syslog.LogPartsChannel)
	out := make(syslog.LogPartsChannel, 1000)

	ring := NewRingBuffer(in, out)
	go ring.Run()

	handler := syslog.NewChannelHandler(in)

	server := syslog.NewServer()
	server.SetFormat(syslog.RFC5424)
	server.SetHandler(handler)

	if err := server.ListenTCP(":8081"); err != nil {
		log.Fatal(err)
	}

	if err := server.Boot(); err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", httpHandler(out))
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func httpHandler(c syslog.LogPartsChannel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		select {
		case log := <-c:
			fmt.Fprintln(w, log)
		default:
			fmt.Fprintln(w, "no logs available")
		}
	}
}
