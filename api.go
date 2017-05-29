package main

type message struct {
	TimeNano int64     `json:"time"`
	Tag      string    `json:"tag"`
	Values   []float64 `json:"values"`
}

// APIResponse is the struct for the client request
type APIResponse struct {
	TagName string  `json:"tagName"`
	Start   int64   `json:"start"`
	End     int64   `json:"end"`
	Samples Samples `json:"samples"`
}

type sample struct {
	Time   int64     `json:"time"`
	Values []float64 `json:"values"`
}

// Samples holds samples
type Samples []sample

// Len is
func (slice Samples) Len() int {
	return len(slice)
}

// Less is
func (slice Samples) Less(i, j int) bool {
	return slice[i].Time < slice[j].Time
}

// Swap is
func (slice Samples) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}
