// Package history stores check results and provides a web interface to view them
package history

import (
	_ "embed"
	"net/http"
	"strconv"
	"sync"
	"text/template"
	"time"

	"golang.org/x/exp/slog"

	"github.com/boyvinall/openstack-check-exporter/pkg/checker"
)

var (
	//go:embed index.html.tpl
	indexTemplate string

	//go:embed detail.tpl
	detailTemplate string
)

// History stores check results and provides a web interface to view them.
// Similar to the web ui provided by the prometheus blackbox exporter.
type History struct {
	lock     sync.Mutex
	maxCount int
	id       uint64
	results  []result
	index    *template.Template
	detail   *template.Template
}

type result struct {
	ID uint64
	*checker.CheckResult
}

// New creates a new History instance
func New(maxCount int) (*History, error) {
	funcMap := template.FuncMap{
		"width": func(d time.Duration) uint64 {
			widthOneSecond := 30 // px
			return uint64((d * time.Duration(widthOneSecond)) / time.Second)
		},
		"duration": func(d time.Duration) time.Duration {
			return d.Round(time.Millisecond)
		},
	}
	index, err := template.New("index").Funcs(funcMap).Parse(indexTemplate)
	if err != nil {
		return nil, err
	}
	detail, err := template.New("detail").Parse(detailTemplate)
	if err != nil {
		return nil, err
	}
	return &History{
		index:    index,
		detail:   detail,
		maxCount: maxCount,
	}, nil
}

// Append adds a new check result to the history
func (h *History) Append(r checker.CheckResult) {
	h.lock.Lock()
	defer h.lock.Unlock()
	h.results = append([]result{
		{
			ID:          h.id,
			CheckResult: &r,
		},
	}, h.results...)
	h.id++
}

// Trim removes the oldest check results if the history is longer than maxCount
func (h *History) Trim() {
	h.lock.Lock()
	defer h.lock.Unlock()
	if len(h.results) > h.maxCount {
		h.results = h.results[:h.maxCount]
	}
}

// ShowList displays the list of check results in a web browser
func (h *History) ShowList(w http.ResponseWriter, r *http.Request) {
	h.lock.Lock()
	defer h.lock.Unlock()

	var results []result
	if checkname := r.URL.Query().Get("name"); checkname == "" {
		results = h.results
	} else {
		for i := range h.results {
			if h.results[i].Name == checkname {
				results = append(results, h.results[i])
			}
		}
	}
	err := h.index.Execute(w, results)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("unable to execute template", "error", err)
		return
	}
}

// ShowDetail displays the details of a single check result in a web browser
func (h *History) ShowDetail(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "" {
		h.ShowList(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	id, err := strconv.ParseUint(r.URL.Path, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	h.lock.Lock()
	defer h.lock.Unlock()
	for _, r := range h.results {
		if r.ID == id {
			err = h.detail.Execute(w, r)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				slog.Error("unable to execute template", "error", err)
			}
			return
		}
	}
}
