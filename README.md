## Usage example

```go
package main

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/nats-io/nats.go"

	"github.com/welteki/natsrr"
)

func main() {
	url := nats.DefaultURL

	nc, _ := nats.Connect(url)
	defer nc.Drain()

	mux := natsrr.NewSubjectMux()

	mux.HandleFunc("greet.joe", func(r natsrr.Responder, msg *nats.Msg) {
		r.Respond([]byte("Hello joe"))
	})

	mux.HandleFunc("greet.sue", func(r natsrr.Responder, msg *nats.Msg) {
		r.SetStatus(http.StatusAccepted)
		r.Respond([]byte("Hello sue"))
	})

	sub, _ := nc.Subscribe("greet.*", mux.MsgHandler())

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	<-sigs

	sub.Unsubscribe()
}
```

### Send requests with the nats CLI

```bash
$ nats req 'greet.joe' ""

11:58:54 Sending request on "greet.joe"
11:58:54 Received with rtt 251.976µs
11:58:54 Status: 200

Hello joe
```

```bash
$ nats req 'greet.sue' ""

11:59:42 Sending request on "greet.sue"
11:59:42 Received with rtt 238.851µs
11:59:42 Status: 202

Hello sue
```

```bash
$ nats req 'greet.josh' ""

11:56:32 Sending request on "greet.josh"
11:56:32 Received with rtt 371.151µs
11:56:32 Description: No messages
11:56:32 Status: 404
```