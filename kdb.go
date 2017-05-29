package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	kdb "github.com/sv/kdbgo"
)

// KDB provides a struct to call `tp` and `hdb` connections
// also provides a `batch chan` for `saveBatch`
// implements `Database` interfaces
type KDB struct {
	tp  *kdb.KDBConn
	hdb *kdb.KDBConn
	out chan []*kdb.K
}

// connects to `tp` and `hdb` instances and initializes a `db.out` batch
// channel for `db.saveBatch()`
func getKDB() Database {
	tpHost := os.Getenv("TP_HOST")

	if tpHost == "" {
		tpHost = "127.0.0.1"
	}

	tpPort, err := strconv.Atoi(os.Getenv("TP_PORT"))

	if err != nil {
		tpPort = 6012
	}

	hdbHost := os.Getenv("HDB_HOST")

	if hdbHost == "" {
		hdbHost = "127.0.0.1"
	}

	hdbPort, err := strconv.Atoi(os.Getenv("HDB_PORT"))

	if err != nil {
		hdbPort = 6013
	}

	var tp, hdb *kdb.KDBConn

	if tp, err = kdb.DialKDB(tpHost, tpPort, ""); err != nil {
		log.Panicf("!> Failed to connect to tp: %s:%d, %v", tpHost, tpPort, err)
	} else {
		log.Printf("> Connected to tp: %s:%d", tpHost, tpPort)
	}

	if hdb, err = kdb.DialKDB(hdbHost, hdbPort, ""); err != nil {
		log.Panicf("Failed to connect to hdb: %s:%d, %v", hdbHost, hdbPort, err)
	} else {
		log.Printf("> Connected to hdb: %s:%d", hdbHost, hdbPort)
	}

	return KDB{tp, hdb, make(chan []*kdb.K, 5)}
}

// used in tests: gets last entry in the provided interval from the DB
func (db KDB) getIntervalSample(tag string, start int64, end int64) (sample, error) {
	q := fmt.Sprintf("-1#select from t where int=`int$`sym$`%s,ts > %d,ts <= %d", tag, start, end)

	log.Printf("> kdb.getIntervalSample: %v", q)

	res, err := db.q(q)

	if err != nil {
		return sample{}, err
	}

	// d := res.Data.(kdb.Dict)
	d := res.Data.(kdb.Table)

	log.Printf("> values: %v\n\n", d.Data[3].Data)
	log.Printf("> values: %v\n\n", d.Data[3].Data.([]*kdb.K)[0].Data.([]float64))

	ts := d.Data[2].Data.([]int64)
	values := d.Data[3].Data.([]*kdb.K)

	return sample{ts[0], values[0].Data.([]float64)}, nil
}

// calls downsampling function on hdb
func (db KDB) getSeries(tag string, start int64, end int64) (APIResponse, error) {
	// log.Printf("> kdb.getSeries %s, %d, %d\n", tag, start, end)
	// s := time.Now()

	var samples Samples

	q := fmt.Sprintf(".P.downsample_tag[`%s; %d; %d]", tag, start, end)

	res, err := db.q(q)

	if err != nil {
		log.Printf("!> Query failed: %v \n%s\n\n", err, q)
		return APIResponse{}, err
	}

	d := res.Data.(kdb.Table)
	ts := d.Data[1].Data.([]int64)
	values := d.Data[3].Data.([]*kdb.K)

	for i := range ts {
		vals, ok := values[i].Data.([]float64)

		if !ok {
			continue
		}

		samples = append(samples, sample{ts[i], vals})
	}

	// log.Printf("> %v getSeries(%s, %d, %d)\n", time.Now().Sub(s), tag, start, end)

	return APIResponse{tag, start, end, samples}, nil
}

// queries on `tp`, not used atm
func (db KDB) query(q string) error {
	_, err := db.tp.Call(q)

	if err != nil {
		return err
	}

	return nil
}

// client queries on `hdb`
func (db KDB) q(q string) (*kdb.K, error) {
	// log.Printf("> kdb.q: %v", q)
	return db.hdb.Call(q)
}

func (db KDB) saveBatch() {
	for rows := range db.out {
		s := time.Now()

		log.Printf("> %v %d --> saveBatch: \n", time.Now(), len(rows))

		if _, err := db.tp.Call(".P.tp_add", kdb.NewList(rows...)); err != nil {
			log.Printf("!!!!!!> can't upsert to .tmp.n: %s, returning to db.outr", err)
			db.out <- rows
			continue
		}

		log.Printf("> %v %d <-- saveBatch\n", time.Now().Sub(s), len(rows))
	}
}

// starts 2 goroutines:
// - one reading from incoming messages `msgChan` channel,
// batching them once per predefined tick interval.
// - another is batch sender, getting batches from `d.out` 'batch chan'
// returns msgChan, so it can be used by saveHandler API
func (db KDB) startQueueConsumer() chan Msg {
	// channel of incoming parsed messages, arriving from saveHandler
	msgChan := make(chan Msg, 100000)

	timer := time.Tick(time.Second)

	// start backgroup batch saver, reading from `db.out` channel
	// add more here to parallelize batch inserts into DB
	go db.saveBatch()

	var rowBatch []*kdb.K

	// launch a goroutine, that adds each incoming message from msgChan
	// to a batch in DB specific format, and sends it to db.out once in
	// timer.Tick
	go func() {
		for {
			select {
			case <-timer:
				if len(rowBatch) == 0 {
					fmt.Printf(".")
					continue
				}

				db.out <- rowBatch

				log.Printf("> %v %d -> saveBatch | batches: %d\n", time.Now(), len(rowBatch), len(db.out))

				rowBatch = []*kdb.K{}
			case m := <-msgChan:
				rowBatch = append(rowBatch, kdb.NewList(kdb.Symbol(m.Tag), kdb.Long(m.Time), kdb.Atom(kdb.KF, m.Values)))
			}
		}
	}()

	return msgChan
}
