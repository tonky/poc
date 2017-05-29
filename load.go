package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/mailru/easyjson"
	"github.com/valyala/fasthttp"
)

// HTTPLoader send load over http
type HTTPLoader struct {
	url      string
	workers  *sync.WaitGroup
	messages chan Msg
	counter  int
	numTags  int
	interval time.Duration
}

func postWorker(messages <-chan Msg, wg *sync.WaitGroup, urls string) {
	defer wg.Done()

	u, err := url.Parse(urls)

	if err != nil {
		log.Fatal(err)
	}

	c := &fasthttp.PipelineClient{
		Addr: u.Host,
		// MaxConns:           1,
		MaxPendingRequests: 100000,
	}

	go func() {
		for {
			log.Printf("> post worker pending requests: %d", c.PendingRequests())
			time.Sleep(time.Second)
		}
	}()

	// req := fasthttp.AcquireRequest()
	// req.Header.SetMethod("POST")
	// req.SetRequestURI(urls)

	for msg := range messages {
		go func() {
			req := fasthttp.AcquireRequest()
			req.Header.SetMethod("POST")
			req.SetRequestURI(urls)
			// resp := fasthttp.AcquireResponse()
			// log.Println("> postWorker url: ", url)

			// resp := fasthttp.AcquireResponse()

			jsonMsg, _ := easyjson.Marshal(msg)

			req.SetBody(jsonMsg)

			err := c.Do(req, nil)

			if err != nil {
				log.Printf("> pending requests: %d", c.PendingRequests())
				log.Fatalf("!> sendPostMessage error: %v\n", err)
			}

		}()
	}

	for c.PendingRequests() > 0 {
		log.Printf("> mesages done, sleeping while pending requests: %d", c.PendingRequests())
		time.Sleep(50 * time.Millisecond)
	}

	time.Sleep(50 * time.Millisecond)

	log.Printf("> post worker closed messages: %d, done witn requests: %v\n", len(messages), c)
}

func newHTTPLoader(url string, numTags int) HTTPLoader {
	if url == "" {
		log.Fatal("! empty url for HTTPLoader")
	}

	postMessages := make(chan Msg, 200000)

	// plenty enough for 200K+ pipelined requests
	numPostWorkers := runtime.NumCPU()

	var wg sync.WaitGroup

	for i := 0; i < numPostWorkers; i++ {
		wg.Add(1)
		go postWorker(postMessages, &wg, url)
	}

	fmt.Printf("> %v, \t added %d 'msgQueue -> POST' workers \n", time.Now(), numPostWorkers)

	return HTTPLoader{url, &wg, postMessages, 0, numTags, time.Hour * 24}
}

func (l *HTTPLoader) add(amt int) time.Duration {
	start := time.Now()
	s := start.UnixNano()

	for i := 0; i < amt; i++ {
		l.messages <- randMsg(s+int64(i), fmt.Sprintf("t%d", i%l.numTags))
	}

	doneIn := time.Now().Sub(start)

	l.counter += amt

	fmt.Printf("> %v, \t %v \t HTTPLoader.add()ed %d messages, queue %d\n", time.Now(), doneIn, amt, len(l.messages))

	return doneIn
}

func (l *HTTPLoader) addOne() Msg {
	m := randMsg(time.Now().UnixNano(), fmt.Sprintf("t%d", rand.Intn(l.numTags)))

	l.messages <- m

	l.counter++

	log.Printf("> %v added to loader", m)

	return m
}

func (l HTTPLoader) stop() {
	close(l.messages)

	log.Printf("> %v \t stopping loader, closed messages chan\n", time.Now())

	l.workers.Wait()

	log.Printf("> %v \t all workers done\n", time.Now())
}

func sendPostMessage(url string, msg Msg) {
	// fmt.Println("> sending message to url: ", url, msg)
	messageJSON, _ := easyjson.Marshal(msg)

	resp, err := http.Post(url, "application/json; charset=utf-8", bytes.NewBuffer(messageJSON))

	if err != nil {
		fmt.Println(err)
	}

	// log.Printf("> post worker sent %v to %v", msg, url)

	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()
}

func envParams(rateEnv string, durationEnv string, urlEnv string) (int, time.Duration, string) {
	rate := 10000
	duration := time.Second * 10
	url := os.Getenv(urlEnv)

	if r, err := strconv.Atoi(os.Getenv(rateEnv)); err == nil && r > 0 {
		rate = r
	}

	if d, err := strconv.Atoi(os.Getenv(durationEnv)); err == nil && d > 0 {
		duration = time.Second * time.Duration(d)
	}

	return rate, duration, url
}

func apiResponseTimes(url string, start int64, end int64, numTags int, stats map[string]int) {
	// running both server and load tests on single machine puts too much strain
	// on the system right when messages are send/received
	// wait a bit before checking api response times to gen an accurate
	// representation of client api
	time.Sleep(300 * time.Millisecond)

	calls := 10

	responseTimes := make(chan time.Duration, calls)

	for i := 0; i < calls; i++ {
		callStart := time.Now()

		tag := fmt.Sprintf("t%d", rand.Intn(numTags))

		_, err := getData(url+"/api", start, end, tag)

		if err != nil {
			fmt.Println("!> expected API response for: ", url+"/api", tag, start, end)
			return
		}

		callEnd := time.Now().Sub(callStart)

		fmt.Printf("> API responded in: %v\n", callEnd)

		responseTimes <- callEnd
	}

	close(responseTimes)

	for rt := range responseTimes {
		if rt > 99*time.Millisecond {
			stats["100+"]++
		}

		if 50*time.Millisecond <= rt && rt <= 99*time.Millisecond {
			stats["99"]++
		}

		if 10*time.Millisecond <= rt && rt < 50*time.Millisecond {
			stats["50"]++
		}

		if rt < 10*time.Millisecond {
			stats["10"]++
		}
	}
}

func getData(url string, start int64, end int64, tag string) (APIResponse, error) {
	var data APIResponse

	fullURL := fmt.Sprintf("%s?start=%d&end=%d&tag=%s", url, start, end, tag)

	c := &fasthttp.Client{}

	code, body, err := c.Get(nil, fullURL)

	if err != nil || code != 200 {
		log.Fatalf("!> error getting API response from %v, %v, %v, %v ", fullURL, start, end, tag)
		return data, err
	}

	if err := json.Unmarshal(body, &data); err != nil {
		log.Printf("!> error unmarshalling API response body to JSON: %s", body)
		return data, err
	}

	return data, nil
}

func randMsg(t int64, tag string) Msg {
	v := float64(t) / 10e12

	return Msg{t, tag, []float64{v, v, v, v, v, v, v, v, v, v}}
}
