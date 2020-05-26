package progress

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	URIProgress    = "/progress"
	URILogs        = "/log"
	URIDeliverable = "/deliverable"
)

var responseBody *Status

type Status struct {
	Complete              bool          `json:"complete"`
	CompletedSuccessfully bool          `json:"completedSuccessfully"`
	Messages              []StatusEvent `json:"messages"`
}

type DeliverableInfo struct {
	Url     string `json:"url"`
	FileExt string `json:"file_extension"`
}

func UpdateProgressComplete(complete bool) {
	responseBody.Complete = complete
}

func UpdateProgressCompletedSuccessfully(completedSuccessfully bool) {
	responseBody.CompletedSuccessfully = completedSuccessfully
}

func init() {
	responseBody = new(Status)
	responseBody.Messages = []StatusEvent{}
}

func Serve(logfile string, ip, port string, status Events, fileDeliverables []string) {
	fullURL := url.URL{Scheme: "http", Host: ip + ":" + port, Path: ""}
	fn := func(p *StatusEvent) {
		responseBody.Messages = append(responseBody.Messages, *p)
	}
	if status != nil {
		status.Subscribe(fn)
	}

	http.HandleFunc(URIProgress, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(responseBody)
	})
	fullURL.Path = URIProgress
	status.Publish(&StatusEvent{
		Type:  "progress",
		Msg:   fmt.Sprintf("serving progress at %v", fullURL.String()),
		Level: "debug",
	})
	http.HandleFunc(URILogs, func(w http.ResponseWriter, r *http.Request) {
		logs, _ := ioutil.ReadFile(logfile)
		fmt.Fprintf(w, string(logs))
	})
	fullURL.Path = URILogs
	status.Publish(&StatusEvent{
		Type:  "progress",
		Msg:   fmt.Sprintf("serving file: %v at %v", logfile, fullURL.String()),
		Level: "debug",
	})

	var dv []DeliverableInfo
	for _, elem := range fileDeliverables {
		var f string
		if elem != "" {
			f = elem
			base := strings.TrimSuffix(filepath.Base(elem), filepath.Ext(elem))
			uri := fmt.Sprintf("%s/%s", URIDeliverable, base)
			http.HandleFunc(uri, func(w http.ResponseWriter, r *http.Request) {
				file, _ := ioutil.ReadFile(f)
				fmt.Fprintf(w, string(file))
			})
			dv = append(dv, DeliverableInfo{
				Url:     uri,
				FileExt: filepath.Ext(elem),
			})
			fullURL.Path = uri
			status.Publish(&StatusEvent{
				Type:  "progress",
				Msg:   fmt.Sprintf("serving file: %v at %v", elem, fullURL.String()),
				Level: "debug",
			})
		}
	}
	http.HandleFunc(URIDeliverable, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(dv)
	})
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func DownloadTxtFile(url string, downloadLocation string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error with GET on: %v, err: %v", url, err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed, %v", resp.StatusCode)
	}
	responseData, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	err = ioutil.WriteFile(downloadLocation, responseData, 0644)
	if err != nil {
		return fmt.Errorf("error writing file to disk: %v, err: %v", downloadLocation, err)
	}
	return nil
}
