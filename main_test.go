package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/labstack/echo"
	"github.com/mailru/easyjson"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

type mockDB struct {
	interval time.Duration
}

func (mdb mockDB) saveBatch() {
	return
}

func (mdb mockDB) startQueueConsumer() chan Msg {
	return make(chan Msg, 1)
}

func (mdb mockDB) getIntervalSample(string, int64, int64) (sample, error) {
	return sample{}, nil
}

func (mdb mockDB) query(string) error {
	return nil
}

func (mdb mockDB) getSeries(tag string, start int64, end int64) (APIResponse, error) {
	mockSample := sample{1000, []float64{1, 2, 3}}

	return APIResponse{tag, start, end, []sample{mockSample}}, nil
}

func TestGetSeries(t *testing.T) {
	t.Parallel()

	ts := getMockTestServer()

	start := time.Now().Add(-10 * time.Hour).UnixNano()
	end := time.Now().Add(-5 * time.Hour).UnixNano()

	url := fmt.Sprintf("http://test.me/api?start=%d&end=%d&tag=test_tag2", start, end)

	var req fasthttp.Request
	var resp fasthttp.Response
	req.SetRequestURI(url)

	err := ts.c.Do(&req, &resp)

	assert.NoError(t, err)

	body := resp.Body()
	statusCode := resp.StatusCode()

	var data APIResponse

	if e := json.Unmarshal(body, &data); e != nil {
		t.Fatal("error decoding API response", e, body)
	}

	mockSample := sample{1000, []float64{1, 2, 3}}

	expected := APIResponse{"test_tag2", start, end, []sample{mockSample}}

	assert.Equal(t, expected, data, "resp body should match")
	assert.Equal(t, 200, statusCode, "should get a 200")
}

func TestSave(t *testing.T) {
	t.Parallel()

	ts := getMockTestServer()

	url := "http://test.me/save"

	req := fasthttp.AcquireRequest()
	req.Header.SetMethod("POST")

	resp := fasthttp.AcquireResponse()
	req.SetRequestURI(url)

	testMessage := Msg{1000, "test_tag", []float64{1.1}}

	jsonMsg, _ := json.Marshal(testMessage)

	req.AppendBody(jsonMsg)

	err := ts.c.Do(req, resp)

	if err != nil {
		t.Fatalf("!> testSave c.Do error: %v", err)
	}

	statusCode := resp.StatusCode()

	assert.Equal(t, 200, statusCode, "should get a 200")
	assert.Equal(t, testMessage, <-ts.in, "should appear in chan")
}

func TestHealth(t *testing.T) {
	t.Parallel()

	ts := getMockTestServer()

	_, body, _ := ts.c.Get(nil, "http://test.me/health")

	assert.Equal(t, "OK", string(body), "should return 'OK'")
}

func TestMaxEnd(t *testing.T) {
	t.Skip("interferes with load tests")
	t.Parallel()

	ts := getMockTestServer()

	start := time.Now().Add(-9 * time.Hour).UnixNano()
	end := time.Now().Add(-9 * time.Minute).UnixNano()

	path := fmt.Sprintf("?start=%d&end=%d&tag=test_tag", start, end)

	code, _, _ := ts.c.Get(nil, "http://test.me/api"+path)

	assert.Equal(t, 400, code, "'end' should be at least 10 minutes ago")
}

func TestMinStart(t *testing.T) {
	t.Parallel()

	ts := getMockTestServer()

	start := time.Now().Add(-25 * time.Hour).UnixNano()
	end := time.Now().Add(-9 * time.Hour).UnixNano()

	path := fmt.Sprintf("?start=%d&end=%d&tag=test_tag", start, end)

	code, _, _ := ts.c.Get(nil, "http://test.me/api"+path)

	assert.Equal(t, 400, code, "'start' should be at maximum 24 hours ago")
}

func BenchmarkEasyJSON(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.POST("/bench", func(c *gin.Context) {
		var m Msg

		body, ioerr := ioutil.ReadAll(c.Request.Body)

		if ioerr != nil {
			log.Panic("> error reading body", ioerr)
		}

		if err := easyjson.Unmarshal(body, &m); err != nil {
			log.Panic("> error decoding json", err)
		}

		c.String(200, m.Tag)
	})

	jsonMsg, _ := json.Marshal(Msg{time.Now().UnixNano(), "tag", []float64{1.2, 2.3, 3.4, 4.5, 5.1, 6.89, 7.0, 8.12, 9.99, 10.10}})

	w := httptest.NewRecorder()

	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "/bench", bytes.NewReader(jsonMsg))

		router.ServeHTTP(w, req)
	}
}

func BenchmarkGinJSON(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.POST("/bench", func(c *gin.Context) {
		var m Msg

		if err := c.BindJSON(&m); err != nil {
			log.Print(">>> can't parse incoming message: ", err, c.Request, c.Request.Body)
		}

		c.String(200, m.Tag)
	})

	jsonMsg, _ := json.Marshal(Msg{time.Now().UnixNano(), "tag", []float64{1.2, 2.3, 3.4, 4.5, 5.1, 6.89, 7.0, 8.12, 9.99, 10.10}})
	// fmt.Printf("> %+q\n", jsonMsg)

	w := httptest.NewRecorder()

	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "/bench", bytes.NewReader(jsonMsg))

		router.ServeHTTP(w, req)
	}
}

func BenchmarkGJSON(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	gjson.UnmarshalValidationEnabled(false)

	router := gin.New()
	router.POST("/bench", func(c *gin.Context) {
		var m Msg

		body, ioerr := ioutil.ReadAll(c.Request.Body)

		if ioerr != nil {
			log.Panic("> error reading body", ioerr)
		}

		if err := gjson.Unmarshal(body, &m); err != nil {
			log.Panic("> error decoding json", err)
		}

		c.String(200, m.Tag)
	})

	jsonMsg, _ := json.Marshal(Msg{time.Now().UnixNano(), "tag", []float64{1.2, 2.3, 3.4, 4.5, 5.1, 6.89, 7.0, 8.12, 9.99, 10.10}})
	// fmt.Printf("> %+q\n", jsonMsg)

	w := httptest.NewRecorder()

	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "/bench", bytes.NewReader(jsonMsg))

		router.ServeHTTP(w, req)
	}
}

func BenchmarkEchoJSON(b *testing.B) {
	jsonMsg, _ := json.Marshal(Msg{time.Now().UnixNano(), "tag", []float64{1.2, 2.3, 3.4, 4.5, 5.1, 6.89, 7.0, 8.12, 9.99, 10.10}})

	e := echo.New()

	e.POST("/bench", func(c echo.Context) (err error) {
		var m Msg

		if err = c.Bind(m); err != nil {
			return
		}

		return c.JSON(http.StatusOK, m.Tag)
	})

	w := httptest.NewRecorder()

	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "/bench", bytes.NewReader(jsonMsg))

		e.ServeHTTP(w, req)
	}
}

func BenchmarkJSON(b *testing.B) {
	var m Msg
	jsonMsg, _ := json.Marshal(Msg{time.Now().UnixNano(), "tag", []float64{1.2, 2.3, 3.4, 4.5, 5.1, 6.89, 7.0, 8.12, 9.99, 10.10}})

	for i := 0; i < b.N; i++ {

		if err := json.Unmarshal([]byte(jsonMsg), &m); err != nil {
			log.Panic("> error decoding json", err)
		}
	}
}

func BenchmarkGJSON_(b *testing.B) {
	var m Msg
	jsonMsg, _ := json.Marshal(Msg{time.Now().UnixNano(), "tag", []float64{1.2, 2.3, 3.4, 4.5, 5.1, 6.89, 7.0, 8.12, 9.99, 10.10}})

	for i := 0; i < b.N; i++ {

		if err := gjson.Unmarshal([]byte(jsonMsg), &m); err != nil {
			log.Panic("> error decoding json", err)
		}
	}
}

func BenchmarkEJSON_(b *testing.B) {
	var m Msg
	jsonMsg, _ := json.Marshal(Msg{time.Now().UnixNano(), "tag", []float64{1.2, 2.3, 3.4, 4.5, 5.1, 6.89, 7.0, 8.12, 9.99, 10.10}})

	for i := 0; i < b.N; i++ {

		if err := easyjson.Unmarshal([]byte(jsonMsg), &m); err != nil {
			log.Panic("> error decoding json", err)
		}
	}
}

func getMockTestServer() testData {
	db := mockDB{}

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
