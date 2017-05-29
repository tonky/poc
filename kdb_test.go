package main

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"testing"
	"time"
)

func kdbInit() Database {
	rand.Seed(time.Now().UnixNano())

	db := getKDB()

	// qEpoch := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
	// log.Printf("> qEpoch ns: %v\n", qEpoch.UnixNano())

	// interval := end.Sub(start).Nanoseconds() / 100
	// .t.interval:{`long$((`timestamp$y) - (`timestamp$x)) % 100}
	// select by .t.interval[start;end] xbar ts from tt where tag=`tag4783,ts>2017.04.26,ts<=2017.04.27
	// tt := kdb.NewTable([]string{"`g#`symbol$()", "`s#`timestamp$()", "`float$()"}, []*kdb.K{})
	// log.Printf("> tt: %s %v %q", tt, tt, tt)

	qInit := []string{
		"delete t from `.",
		"t: .P.gen_t[]",
		// "interval: {`long$((`timestamp$y) - (`timestamp$x)) % 100}",
		// "group_by: {[tbl;t;s;e] select by interval[s;e] xbar ts from tbl where tag=t,ts>s,ts<=e}",
	}

	// q := "delete tt from `.; tt: ([] tag:`g#`symbol$(); ts:`s#`timestamp$(); val:`float$())"
	q := strings.Join(qInit, "; ")

	if err := db.query(q); err != nil {
		log.Fatal("!> init query failed:", q, err)
	}

	return db
}

/*
func addRows(con *kdb.KDBConn, numRows int, numTags int) {
	for i := 0; i < numRows; i++ {
		tag := fmt.Sprintf("t%d", rand.Intn(numTags))

		q := fmt.Sprintf("`t upsert (`%s;`timestamp$%d;%f)", tag, kn(time.Now()), rand.Float64())

		if _, err := con.Call(q); err != nil {
			log.Fatal("Query failed:", err)
		}
	}
}

func TestTableRW(t *testing.T) {
	numTags := 10000
	numRows := 1000

	db := kdbInit()

	now := time.Now()
	// addRows(con, numRows, numTags)
	// fmt.Printf("> added %d rows in %v\n", numRows, time.Now().Sub(now))

	msgs := getRowsBulk(numRows, numTags)

	err := db.saveBatch(msgs)

	if err != nil {
		t.Fatal("failed saving batch")
	}

	now = time.Now().UTC()
	val := rand.Float64()

	q := fmt.Sprintf(".tmp.r: ([] tag:%s; ts:enlist `timestamp$%d; val:enlist %g); .P.upsert_all[.tmp.r]", "`t5555", kn(now), val)

	if _, err := con.Call(q); err != nil {
		t.Fatal("=Query failed: ", q, err)
	} else {
		fmt.Printf("> added test row: %v", q)
	}

	fmt.Printf("> q upsert: %v\n", q)

	if _, err := con.Call("\\l /tmp/db"); err != nil {
		t.Fatal("!> query failed: \\l /tmp/db", q, err)
	} else {
		fmt.Printf("> reloaded t: %v\n", q)
	}

	q = fmt.Sprintf(".P.downsample_aj_notag[select from t where int=`int$`sym$`t5555; `timestamp$%d;`timestamp$%d]", kn(now.Add(-24*time.Hour)), kn(now))

	res, err := con.Call(q)

	if err != nil {
		t.Fatal("-Query failed: ", q, err)
	}

	// fmt.Printf("> q: %v\n> res: %v\n", q, res)

	d := res.Data.(kdb.Table)
	ts := d.Data[0].Data.([]time.Time)
	values := d.Data[3].Data.([]float64)

	// d := res.Data.(kdb.Dict)
	// ts := d.Key.Data.(kdb.Table).Data[0].Data.([]time.Time)
	// values := d.Value.Data.(kdb.Table).Data[2].Data.([]float64)

	// fmt.Printf("> table res: %v\n", d)

	// fmt.Printf("Q: %v, result: %v, %v, %+v, %+q\n", q, res.Type, res.Attr, ts, values)
	// fmt.Printf("%v => %v\n", d, d.Data[0].Data.([]time.Time)[0])
		for i := range ts {
			fmt.Printf("> %v | ts: %v, value: %v\n", i, ts[i], values[i])
		}

	if ts[len(ts)-1] != now || values[len(ts)-1] != val {
		// if values[99] != val {
		t.Fatalf("wanted: %v | %v, got: %v | %v\n", now, val, ts[len(ts)-1], values[len(ts)-1])
	}

}
*/

func TestBatchSave(t *testing.T) {
	t.Skip("integration")
	// db := getKDB()
	// in := db.startQueueConsumer()
	td := getTestServer()
	amt := 1000000
	/*
		go func() {
			s := &fasthttp.Server{
				Handler: fhMux(db, in),
			}

			ln := fasthttputil.NewInmemoryListener()

			if err := s.Serve(ln); err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
		}()
	*/
	/*
		router := gin.New()
		gin.SetMode(gin.ReleaseMode)
		router.POST("/save", saveHandler(in))
	*/
	s := time.Now()

	for i := 0; i < amt; i++ {
		tag := fmt.Sprintf("t%d", rand.Intn(10000))

		values := []float64{rand.Float64(), rand.Float64(), rand.Float64(), rand.Float64(),
			rand.Float64(), rand.Float64(), rand.Float64(), rand.Float64(), rand.Float64(), rand.Float64()}

		td.in <- Msg{time.Now().UnixNano(), tag, values}

	}
	/*
		go func() {
			w := httptest.NewRecorder()

			for i := 0; i < amt/2; i++ {
				tag := fmt.Sprintf("t%d", rand.Intn(10000))

				values := []float64{rand.Float64(), rand.Float64(), rand.Float64(), rand.Float64(),
					rand.Float64(), rand.Float64(), rand.Float64(), rand.Float64(), rand.Float64(), rand.Float64()}

				jsonMsg, _ := json.Marshal(Msg{time.Now(), tag, values})

				req, err := http.NewRequest("POST", "/save", bytes.NewReader(jsonMsg))

				if err != nil {
					fmt.Printf("!> error creating request", err)
					return
				}

				router.ServeHTTP(w, req)
			}

		}()

		w := httptest.NewRecorder()

		for i := amt / 2; i < amt; i++ {
			tag := fmt.Sprintf("t%d", rand.Intn(10000))

			values := []float64{rand.Float64(), rand.Float64(), rand.Float64(), rand.Float64(),
				rand.Float64(), rand.Float64(), rand.Float64(), rand.Float64(), rand.Float64(), rand.Float64()}

			jsonMsg, _ := json.Marshal(Msg{time.Now(), tag, values})

			req, err := http.NewRequest("POST", "/save", bytes.NewReader(jsonMsg))

			if err != nil {
				fmt.Printf("!> error creating request", err)
				return
			}

			router.ServeHTTP(w, req)
		}
	*/
	log.Printf("> %v sent all Msg\n", time.Now().Sub(s))

	time.Sleep(1 * time.Second)
}

/*
func TestOutQueue(t *testing.T) {
	amt := 1000000
	in := make(chan Msg, amt)
	out := make(chan *kdb.K, amt)

	router := gin.New()
	gin.SetMode(gin.ReleaseMode)
	router.POST("/save", saveHandler(in))

	jsonMsg, _ := json.Marshal(Msg{time.Now().UnixNano(), "tag", []float64{1.2, 2.3, 3.4, 4.5, 5.1, 6.89, 7.0, 8.12, 9.99, 10.10}})
	fmt.Printf("> %q\n", jsonMsg)

	s := time.Now()

	w := httptest.NewRecorder()

	for i := 0; i < amt; i++ {
		req, err := http.NewRequest("POST", "/save", bytes.NewReader(jsonMsg))

		if err != nil {
			fmt.Printf("!> error creating request", err)
			return
		}

		router.ServeHTTP(w, req)
	}

	for len(in) < amt {
		log.Println("> sleeping whole len(out) is < amt", len(out), amt)
		time.Sleep(50 * time.Millisecond)
	}

	log.Printf("> %v sent all Msg\n", time.Now().Sub(s))

	close(in)

	s = time.Now()

	for m := range in {
		// log.Println("> found in: ", m)
		out <- kdb.NewList(kdb.Symbol(m.Tag), kdb.Long(m.Time), kdb.Atom(kdb.KF, m.Values))
		// log.Println("> sent to out")
	}

	log.Printf("> %v converted all Msg to kdb\n", time.Now().Sub(s))

	// fmt.Printf("> %v %d -> POST /save -> %d | %d\n", time.Now().Sub(s), amt, len(in), len(out))

	s = time.Now()

	var rows []*kdb.K

	for len(out) > 0 {
		rows = append(rows, <-out)
	}

	log.Printf("> %v assembled %d rows for Q payload\n", time.Now().Sub(s), len(rows))

	s = time.Now()

	tp, err := kdb.DialKDB("localhost", 6012, "")

	if err != nil {
		log.Panic("Failed to connect to tp on port: ", 6012, err)
	}

	log.Printf("> %v connected to KDB\n", time.Now().Sub(s))

	if _, err := tp.Call(".P.tp_add", kdb.NewList(rows...)); err != nil {
		// if _, err := tp.Call(".tmp.n", kdb.NewList(rows...)); err != nil {
		// if _, err := db.conn.Call("til", kdb.Int(10)); err != nil {
		log.Panicf("> can't upsert to .P.tp_add: %s ", err)
	} else {
		log.Printf("> %v sent %d rows", time.Now().Sub(s), len(rows))
	}

}
*/
