package main

import (
	"fmt"
	"log"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

func addMessagesInterval(start int64, end int64, amt int64, tags int, msgs chan Msg) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	step := (end - start) / amt

	log.Printf("> start: %d, end: %d, step: %d", start, end, step)

	for i := start; i <= end; i += step {
		msgs <- randMsg(i, fmt.Sprintf("t%d", r.Intn(tags)))
	}

	fmt.Printf("> added %d messages for %v tags with step %v, msgChan: %d\n", amt, tags, step, len(msgs))
}

func TestPopulateOneTag24h(t *testing.T) {
	t.Skip("too slow atm")
	period := 24 * time.Hour
	amount := int64(86400)
	interval := period.Nanoseconds() / 100

	// now := time.Now()
	var start int64 = 1000000000000000000
	end := start + period.Nanoseconds()
	tag := "t0"

	td := getTestServer()

	fmt.Printf("> messages %d\n", len(td.in))

	addMessagesInterval(start, end, amount, 1, td.in)

	// wait for send chan to be done
	for len(td.in) > 0 {
		fmt.Printf("> still in db.in: %d\n", len(td.in))
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("> all messages out, sleeping for 2 seconds")
	time.Sleep(time.Millisecond * 3000)

	want0, _ := td.db.getIntervalSample(tag, start, start+interval)
	want99, _ := td.db.getIntervalSample(tag, end-interval, end)

	res, _ := getData(td.apiURL(), start, end, tag)
	// res, _ := db.getSeries(tag, start, end)
	fmt.Printf("> res: %v\n", res)

	got0 := res.Samples[0]
	got99 := res.Samples[99]

	if !reflect.DeepEqual(want0, got0) {
		t.Errorf("!> incorrect first sample, \nwanted %v, \ngot    %v", want0, got0)
	}

	if !reflect.DeepEqual(want99, got99) {
		t.Errorf("!> incorrect last sample, wanted %v, got %v", want99, got99)
	}
}

/*
func TestAPIResponse(t *testing.T) {
	td := getTestServer()
	defer td.stop()

	start := time.Now().Add(-time.Minute * 30).UnixNano()
	end := start + time.Duration(20*time.Minute).Nanoseconds()

	tag := "test_tag"

	msg := Msg{end, tag, []float64{1.2}}

	sendPostMessage(td.saveURL(), msg)

	time.Sleep(2 * time.Second)

	data, err := getData(td.apiURL(), start, end, tag)

	if err != nil {
		t.Fatal("error getting api data")
	}

	if len(data.Samples) == 0 {
		t.Fatalf("expected samples")
	}

	mockSample := sample{data.Samples[0].Time, msg.Values} // rounding error

	expected := APIResponse{tag, start, end, []sample{mockSample}}

	if !reflect.DeepEqual(data, expected) {
		t.Fatalf("Incorrect API response, \nwanted: %+v, \ngot   : %+v", expected, data)
	}
}
*/
