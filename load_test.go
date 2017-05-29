package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

func TestInsertSpeed(t *testing.T) {
	t.Skip("not working yet")

	db := getDB("test")
	numTags := 10000
	// expectedWriteRate := runtime.NumCPU() * batchSize
	expectedWriteRate := 200000

	msgChan := db.startQueueConsumer()
	totalInserted := 0

	for i := 0; i < 40; i++ {
		rate := 10000 + i*5000

		start := time.Now()

		for j := 0; j < rate; j++ {
			msgChan <- randMsg(time.Now().UnixNano(), fmt.Sprintf("t%d", rand.Intn(numTags)))
		}

		for len(msgChan) > 0 {
			time.Sleep(time.Millisecond * 100)
		}

		elapsed := time.Now().Sub(start)

		totalInserted += rate

		if elapsed < time.Millisecond*1250 {
			continue
		}

		fmt.Println("> total inserted: ", totalInserted)

		if rate > expectedWriteRate {
			return
		}

		t.Fatalf("!> wanted wps: %d, got: %d\n", expectedWriteRate, rate)
	}
}

func Test2sLag(t *testing.T) {
	if os.Getenv("URL") == "" {
		t.Skip("skipping test; URL not set")
	}

	rate, duration, url := envParams("RATE", "DURATION", "URL")
	numTags := 10000

	if os.Getenv("URL") == "" {
		t.Skip("skipping test; URL not set")
	}

	loader := newHTTPLoader(url+"/save", numTags)

	end := time.Now().UnixNano()
	start := end - time.Duration(-24*time.Hour).Nanoseconds()

	stats := map[string]int{"10": 0, "50": 0, "99": 0, "100+": 0}

	rand.Seed(end)

	for i := 0.0; i < duration.Seconds(); i++ {
		elapsed := loader.add(rate)

		m := loader.addOne()

		go checkMessageReadable(url, m, t)

		go apiResponseTimes(url, start, end, numTags, stats)

		log.Printf("> waiting %v before generating next batch\n", time.Second-elapsed)

		time.Sleep(time.Second - elapsed)
	}

	loader.stop()

	time.Sleep(2 * time.Second)
	time.Sleep(100 * time.Millisecond)

	if len(loader.messages) > 0 {
		t.Errorf("!> still %d messages in send chan", len(loader.messages))
	}

	fmt.Printf("\n> stats: %v\n\n", stats)

	totalRequests := duration.Seconds() * 10

	pct99 := (totalRequests - float64(stats["100+"])) / totalRequests * 100

	// log.Printf("> totalrequets: %d, pct99: %d\n", totalRequests, pct99)

	if pct99 < 99 {
		t.Errorf("!> wanted 99%% under 100ms, got %v", pct99)
	}

	fmt.Println("finished 1K 2second lag test")
}

func checkMessageReadable(url string, m Msg, t *testing.T) {
	// fmt.Println("> asking for written message: ", url, m)
	<-time.After(time.Second * 2)

	start := m.Time - time.Duration(10*time.Minute).Nanoseconds()

	res, _ := getData(url+"/api", start, m.Time, m.Tag)

	// fmt.Println("> got samples: ", len(res.Samples))

	if len(res.Samples) == 0 {
		t.Fatal("!> expected samples in response!")
	}

	if reflect.DeepEqual(res.Samples[len(res.Samples)-1].Values, m.Values) {
		return
	}

	log.Printf("!> requested: %s/api?tag=%v&start=%d&end=%d\n", url, m.Tag, start, m.Time)
	// log.Printf("!> resp: %v\n", res)

	t.Errorf("> \nwant: %v\ngot : %v\n", m, res.Samples[len(res.Samples)-1])
}

type testData struct {
	ts *fasthttp.Server
	ln net.Listener
	c  *fasthttp.Client
	db Database
	in chan Msg
}

func (td testData) stop() {
	// td.db.stopTicker()
	td.ln.Close()
	close(td.in)

	log.Println("> stopped test server, db ingester ticker and closed messages channel")
}

func (td testData) saveURL() string {
	return fmt.Sprintf("%s://%s/save", td.ln.Addr().Network, td.ln.Addr().String())
}

func (td testData) apiURL() string {
	return fmt.Sprintf("%s://%s/api", td.ln.Addr().Network, td.ln.Addr().String())
}

func getTestServer() testData {
	db := getDB("test")

	in := db.startQueueConsumer()

	s := &fasthttp.Server{
		Handler: fhMux(db, in),
	}

	ln := fasthttputil.NewInmemoryListener()

	go func() {
		if err := s.Serve(ln); err != nil {
			log.Fatalf("unexpected error: %s", err)
		}
	}()

	c := &fasthttp.Client{
		Dial: func(addr string) (net.Conn, error) {
			return ln.Dial()
		},
	}

	return testData{s, ln, c, db, in}
}
