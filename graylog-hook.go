package grayhook

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// BufSize = <value> set before calling NewGraylogHook
// Once the buffer is full, logging will start blocking, waiting for slots to
// be available in the queue.
var BufSize uint = 8192

// GraylogMessage ...
type GraylogMessage struct {
	Version   string                 `json:"version,omitempty"`
	Host      string                 `json:"host,omitempty"`
	Short     string                 `json:"short_message,omitempty"`
	Full      string                 `json:"full_message,omitempty"`
	TimeUnix  time.Time              `json:"timestamp,omitempty"`
	Level     int32                  `json:"level,omitempty"`
	Facility  string                 `json:"facility,omitempty"`
	File      string                 `json:"file,omitempty"`
	Line      int                    `json:"line,omitempty"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
	LogFields map[string]interface{} `json:"log_fields,omitempty"`
}

// GraylogHook is a writer for graylog
type GraylogHook struct {
	graylogAddress string                 // "http://graylog.sdc.com:12201/gelf"
	hostname       string                 // getting by os.Hostname
	facility       string                 // getting by os.Hostname
	extra          map[string]interface{} // will add always
	retries        int                    // number of retry pos (every 10 second)
	buf            chan []byte            // chan for send
	wg             sync.WaitGroup         // wait group for graceful shutdown
	httpClient     *http.Client           // client for post
	Level          logrus.Level
}

// NewGraylogHook creates a Writer
func NewGraylogHook(graylogAddress string, retries int, extra map[string]interface{}, httpClient *http.Client) (*GraylogHook, error) {
	host, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	facility := path.Base(os.Args[0])

	if httpClient == nil {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		httpClient = &http.Client{Transport: tr}
	}

	hook := &GraylogHook{
		graylogAddress: graylogAddress,
		hostname:       host,
		facility:       facility,
		extra:          extra,
		retries:        retries,
		buf:            make(chan []byte, BufSize),
		httpClient:     httpClient,
		Level:          logrus.DebugLevel,
	}

	go hook.fire() // Log in background

	return hook, nil
}

func (hook *GraylogHook) sendEntry(messageBytes []byte) {
	defer hook.wg.Done()

	for i := 0; i < hook.retries; i++ {
		reqPost, err := http.NewRequest("POST", hook.graylogAddress, bytes.NewBuffer(messageBytes))
		if err != nil {
			time.Sleep(10 * time.Second)
			continue
		}

		respPost, err := hook.httpClient.Do(reqPost)
		if err != nil {
			time.Sleep(10 * time.Second)
			continue
		}
		defer respPost.Body.Close()
		break
	}
}

// fire will loop on the 'buf' channel, and write entries to graylog
func (hook *GraylogHook) fire() {
	for {
		messageBytes := <-hook.buf // receive new messageBytes on channel
		hook.sendEntry(messageBytes)
	}
}

//Fire is invoked each time a log is thrown
func (hook *GraylogHook) Fire(entry *logrus.Entry) error {
	fmt.Println(entry.Data)
	grMessage := &GraylogMessage{
		Version: "1.0",
		Host:    hook.hostname,
		Short:   entry.Message,
		// Full:     entry.Data,
		TimeUnix:  time.Now(),
		Level:     setLevel(entry.Level),
		Facility:  hook.facility,
		LogFields: entry.Data,
		Extra:     hook.extra,
	}

	messageBytes, err := json.Marshal(grMessage)
	if err != nil {
		return err
	}

	hook.wg.Add(1)
	hook.buf <- messageBytes

	return nil
}

// Flush - wait until all logs has been send
func (hook *GraylogHook) Flush() {
	hook.wg.Wait()
}

// Levels returns the available logging levels.
func setLevel(logrusLevel logrus.Level) int32 {
	return int32(logrusLevel)
}

// Levels returns the available logging levels.
func (hook *GraylogHook) Levels() []logrus.Level {
	levels := []logrus.Level{}
	for _, level := range logrus.AllLevels {
		if level <= hook.Level {
			levels = append(levels, level)
		}
	}
	return levels
}
