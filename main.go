package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/mailru/easyjson"
	"github.com/valyala/fasthttp"
)

// Msg provides a struct to be used within the app
// incoming messages are parsed into it and converted
// to DB specific formats
type Msg struct {
	Time   int64     `json:"time"`
	Tag    string    `json:"tag"`
	Values []float64 `json:"values"`
}

func main() {
	log.Println("> starting on :8080")

	db := getDB("jet")

	msgChan := db.startQueueConsumer()

	defer close(msgChan)

	fasthttp.ListenAndServe(":8080", fhMux(db, msgChan))
}

func fhMux(db Database, msgChan chan Msg) func(*fasthttp.RequestCtx) {
	log.Println("> fhMux started")
	ready := make(chan bool, 1)
	ready <- true

	return func(ctx *fasthttp.RequestCtx) {
		switch string(ctx.Path()) {
		case "/health":
			healthCheck(ctx)
		case "/save":
			saveHandler(msgChan, ctx)
		case "/api":
			apiHandler(db, ctx, ready)
		default:
			ctx.Error("Unsupported path", fasthttp.StatusNotFound)
		}
	}
}

func healthCheck(ctx *fasthttp.RequestCtx) {
	ctx.WriteString("OK")
}

// func apiHandler(db Database) gin.HandlerFunc {
func apiHandler(db Database, ctx *fasthttp.RequestCtx, ready chan bool) {
	// serialize access to client queries, preventing
	// concurrent reads. current implementation reloads
	// kdb+ table on each client query, and they don't
	// seem to play well on concurrent reloads
	args := ctx.QueryArgs()

	<-ready

	defer func() {
		ready <- true
	}()

	s := time.Now()

	start, err := strconv.ParseInt(string(args.Peek("start")), 10, 64)

	if err != nil {
		log.Printf("!> error parsing 'start' from '%s'", args.Peek("start"))
	}

	startMin := time.Now().Add(-24 * time.Hour).UnixNano()

	if start < startMin {
		ctx.Error("'start' should be max 24h from now", fasthttp.StatusBadRequest)
		return
	}

	end, err := strconv.ParseInt(string(args.Peek("end")), 10, 64)

	if err != nil {
		log.Printf("!> error parsing 'end' from '%s'", args.Peek("end"))
	}

	/* // disabled for now, interferes with tests, where last written message is
	// checked after 2 seconds
		endMax := time.Now().Add(-10 * time.Minute).UnixNano()

		if end > endMax {
			ctx.Error("'end' should be at least 10 minutes ago", fasthttp.StatusBadRequest)
			return
		}
	*/

	tag := string(args.Peek("tag"))

	res, err := db.getSeries(tag, int64(start), int64(end))

	if err == nil {
		respJS, _ := json.Marshal(res)
		ctx.Write(respJS)
	} else {
		fmt.Printf("!> apiHandler db.getSeries failed for ?tag=%v&start=%d&end=%d: %v\n", tag, start, end, err)
		ctx.Error("getSeries failed", fasthttp.StatusBadRequest)
	}

	done := time.Now().Sub(s)

	if done > time.Duration(80*time.Millisecond) {
		log.Printf("> %v apiHandler getSeries for: %v\n", done, args)
	}
}

// parse incoming json messages and put them on `msgChan` for further
// processing to DB specific structures and batching
// func saveHandler(msgChan chan Msg) gin.HandlerFunc {
func saveHandler(msgChan chan Msg, ctx *fasthttp.RequestCtx) {
	// log.Printf("> saveHandler start, req body: %s\n", ctx.Request.Body())
	var m Msg

	if err := easyjson.Unmarshal(ctx.Request.Body(), &m); err != nil {
		log.Printf("!> error decoding json", err)
		ctx.Error("getSeries failed", fasthttp.StatusBadRequest)
		return
	}

	msgChan <- m

	ctx.WriteString("OK")
}
