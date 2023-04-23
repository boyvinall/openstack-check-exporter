package history

import (
	_ "embed"
	"net/http"
	"strconv"
	"sync"
	"text/template"

	"github.com/boyvinall/openstack-check-exporter/pkg/checker"
)

var (
	//go:embed index.html.tpl
	indexTemplate string

	//go:embed detail.tpl
	detailTemplate string
)

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

func New(maxCount int) (*History, error) {
	index, err := template.New("index").Parse(indexTemplate)
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

func (h *History) Append(r *checker.CheckResult) {
	h.lock.Lock()
	defer h.lock.Unlock()
	h.results = append([]result{
		{
			ID:          h.id,
			CheckResult: r,
		},
	}, h.results...)
	h.id++
}

func (h *History) Trim() {
	h.lock.Lock()
	defer h.lock.Unlock()
	if len(h.results) > h.maxCount {
		h.results = h.results[:h.maxCount]
	}
}

func (h *History) ShowList(w http.ResponseWriter, r *http.Request) {
	h.lock.Lock()
	defer h.lock.Unlock()

	h.index.Execute(w, h.results)
}

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
			h.detail.Execute(w, r)
			return
		}
	}
}
