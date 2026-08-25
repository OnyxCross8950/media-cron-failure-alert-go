package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func main() {
	job := flag.String("job", "media-hourly-stream", "scheduled job name")
	stream := flag.String("stream", "nightly-highlights", "media stream name")
	message := flag.String("message", "segment upload exited with status 1", "failure message")
	flag.Parse()

	client, err := NewClient()
	if err != nil {
		fail(err)
	}
	data, err := client.Capture(context.Background(), map[string]string{
		"type":    "scheduled_media_job_failure",
		"job":     *job,
		"stream":  *stream,
		"message": *message,
	})
	if err != nil {
		fail(err)
	}
	var result any
	if err := json.Unmarshal(data, &result); err != nil {
		fail(fmt.Errorf("decode event data: %w", err))
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		fail(fmt.Errorf("encode event data: %w", err))
	}
	fmt.Printf("captured media job failure: %s\n", encoded)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
