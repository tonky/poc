package main

// change here to switch `Database` implementation, that code uses
// see 'getDB' below for candidates
const dbVendor = "kdb"

// Database provides
type Database interface {
	getSeries(tag string, start int64, end int64) (APIResponse, error)
	getIntervalSample(tag string, start int64, end int64) (sample, error)
	saveBatch()
	startQueueConsumer() chan Msg
	query(string) error
}

// currently implemented options are 'kdb' and 'clickhouse'
// see `kdb.go` for implementation and comments
// 'clickhouse' one is experimental
func getDB(dbName string) Database {
	return getKDB()
}
