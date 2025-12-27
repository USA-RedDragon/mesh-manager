package walker

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/USA-RedDragon/mesh-manager/internal/server/api/apimodels"
	"github.com/USA-RedDragon/mesh-manager/internal/walker/concurrentarray"
	"github.com/USA-RedDragon/mesh-manager/internal/walker/http"
	"github.com/puzpuzpuz/xsync/v4"
)
type Task struct {
	Hostname string
	Func     func() (*apimodels.SysinfoResponse, error)
}

type Walker struct {
	client       *http.Client
	responseChan chan *apimodels.SysinfoResponse
	tasks        chan Task
	seen         concurrentarray.ConcurrentArray[string]
	wg           sync.WaitGroup
	TotalCount   *xsync.Counter
}

func NewWalker(timeout time.Duration, retries int, jitter time.Duration) *Walker {
	return &Walker{
		client:       http.NewClient(timeout, retries, jitter),
		responseChan: make(chan *apimodels.SysinfoResponse, 1),
		tasks:        make(chan Task),
		seen:         concurrentarray.ConcurrentArray[string]{},
		wg:           sync.WaitGroup{},
		TotalCount:   xsync.NewCounter(),
	}
}

func (w *Walker) Walk(startingNode string) (chan *apimodels.SysinfoResponse, error) {
	go func() {
		for task := range w.tasks {
			go func() {
				defer w.wg.Done()

				response, err := task.Func()
				if err != nil {
					slog.Error("Error fetching data", "node", task.Hostname, "error", err)
				}
				w.responseChan <- response
			}()
		}
	}()

	resp, err := w.walk(startingNode)
	if err != nil {
		return nil, fmt.Errorf("failed to walk starting node: %w", err)
	}

	w.responseChan <- resp

	go func() {
		w.wg.Wait()
		close(w.tasks)
		close(w.responseChan)
	}()

	return w.responseChan, nil
}

func (w *Walker) walk(node string) (*apimodels.SysinfoResponse, error) {
	url := fmt.Sprintf("http://%s.local.mesh:8080/cgi-bin/sysinfo.json?hosts=1&link_info=1&lqm=1", node)
	resp, err := w.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get sysinfo from node %s: %w", node, err)
	}

	for _, host := range resp.GetHosts() {
		midMatch := regexp.MustCompile(`^mid[0-9]+\.`)
		if (strings.HasPrefix(host.Name, "lan.") && strings.HasSuffix(host.Name, ".local.mesh")) || midMatch.Match([]byte(host.Name)) {
			continue
		}
		if !w.seen.ContainsOrSet(strings.ToUpper(host.Name)) {
			w.wg.Add(1)
			w.TotalCount.Inc()
			go func() {
				w.tasks <- Task{
					Hostname: host.Name,
					Func: func() (*apimodels.SysinfoResponse, error) {
						return w.walk(host.Name)
					},
				}
			}()
		}
	}

	return resp, nil
}