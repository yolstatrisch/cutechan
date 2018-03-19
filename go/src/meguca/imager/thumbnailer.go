package imager

import (
	"meguca/ipc"
)

const (
	// Maximum number of thumbnailer processes executing at the same time.
	THUMB_PROCESSES = 1
)

var (
	jobs = make(chan jobRequest)
)

type jobRequest struct {
	data   []byte
	result chan<- jobResult
}

type jobResult struct {
	thumb *ipc.Thumb
	err   error
}

func worker() {
	for {
		req := <-jobs
		thumb, err := ipc.GetThumbnail(req.data)
		req.result <- jobResult{thumb, err}
	}
}

func requestThumbnail(srcData []byte) (res chan jobResult) {
	res = make(chan jobResult)
	jobs <- jobRequest{srcData, res}
	return
}

// Generate image thumbnail with the specific concurrency level.
func getThumbnail(srcData []byte) (*ipc.Thumb, error) {
	res := <-requestThumbnail(srcData)
	return res.thumb, res.err
}

// Start thumbnailer workers.
func Start() {
	for i := 0; i < THUMB_PROCESSES; i++ {
		go worker()
	}
}
