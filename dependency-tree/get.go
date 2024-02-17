package dependency_tree

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type DependencyTree struct {
	mu                                  sync.Mutex
	wg                                  sync.WaitGroup
	RepositoryChannel                   chan string
	LastTimeDataSentToRepositoryChannel time.Time
	GoModChanel                         chan *GoMod
	LastTimeDataSentToGoModChannel      time.Time
	ErrorChannel                        chan error
	RepositoryToArtifactMap             map[string]*Artifact
	timer                               time.Timer
}

type Artifact struct {
	Name         string
	Version      string
	Dependencies []*Artifact
}

type DependencyTreeFetcher interface {
	Get(c *gin.Context)
}

func NewDependencyTree() DependencyTreeFetcher {
	return &DependencyTree{
		mu:                                  sync.Mutex{},
		wg:                                  sync.WaitGroup{},
		RepositoryChannel:                   make(chan string),
		GoModChanel:                         make(chan *GoMod),
		ErrorChannel:                        make(chan error),
		RepositoryToArtifactMap:             make(map[string]*Artifact),
		LastTimeDataSentToGoModChannel:      time.Now(),
		LastTimeDataSentToRepositoryChannel: time.Now(),
	}
}

func (d *DependencyTree) Get(c *gin.Context) {
	reqBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}
	var body GetBody
	if err = json.Unmarshal(reqBody, &body); err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}

	if body.Url == "" || !d.isValidURL(body.Url) {
		c.JSON(http.StatusBadRequest, "invalid Url")
		return
	}

	resp := d.getDependencyTree(body.Url)
	c.JSON(http.StatusOK, resp)
}

func (d *DependencyTree) isValidURL(rawURL string) bool {
	_, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return false
	}

	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}
	return true
}

func (d *DependencyTree) getDependencyTree(url string) *Artifact {

	d.mu.Lock()
	d.RepositoryToArtifactMap[url] = &Artifact{
		Name:         url,
		Version:      "",
		Dependencies: make([]*Artifact, 0),
	}
	d.mu.Unlock()

	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		d.RepositoryChannel <- url
	}()

	d.wg.Add(1)
	go d.fetch()

	d.wg.Add(1)
	go d.parse()

	d.wg.Add(1)
	go d.errorHandler()

	d.wg.Done()
	d.wg.Wait()
	return d.RepositoryToArtifactMap[url]
}
