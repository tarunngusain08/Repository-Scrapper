package dependency_tree

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type GitHubContent struct {
	Content string `json:"content"`
}

type GetBody struct {
	Url string `json:"Url"`
}

type GoMod struct {
	GoModContent string
	ParentUrl    string
}

// fetchGoMod fetches the content of the go.mod file from the GitHub repository.
func (d *DependencyTree) fetchGoMod(repoURL string) ([]byte, error) {

	// Construct the GitHub API URL to fetch the go.mod file content
	s := strings.Split(repoURL, "/")
	if strings.HasPrefix(repoURL, "http") {
		repoURL = s[3] + "/" + s[4]
	} else if strings.HasPrefix(repoURL, "github.com") {
		repoURL = s[1] + "/" + s[2]
	} else {
		return nil, nil
	}
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/contents/go.mod", repoURL)

	// Make an HTTP GET request to the GitHub API
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("error making HTTP request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	// Parse the JSON response
	var githubContent GitHubContent
	if err = json.Unmarshal(body, &githubContent); err != nil {
		return nil, fmt.Errorf("error parsing JSON response: %v", err)
	}

	// Decode the base64-encoded content of the go.mod file
	goModContent, err := base64.StdEncoding.DecodeString(githubContent.Content)
	if err != nil {
		return nil, fmt.Errorf("error decoding go.mod content: %v", err)
	}
	return goModContent, nil
}

func (d *DependencyTree) fetch() {
	defer d.wg.Done()

	for {
		select {
		case repository, ok := <-d.RepositoryChannel:
			if !ok {
				return // Channel closed, no more data will be sent
			}
			d.wg.Add(1)
			go func(repo string) {
				defer d.wg.Done()
				goModContent, err := d.fetchGoMod(repo)
				if err != nil {
					d.ErrorChannel <- err
					return
				}
				if goModContent != nil && len(goModContent) > 0 {
					goMod := &GoMod{
						GoModContent: string(goModContent),
						ParentUrl:    repo,
					}
					d.GoModChanel <- goMod
				}
				d.mu.Lock()
				if !d.FetchTimeOut.Stop() {
					<-d.FetchTimeOut.C
				}
				d.FetchTimeOut.Reset(5 * time.Second)
				d.mu.Unlock()
			}(repository)
		case <-d.FetchTimeOut.C:
			log.Println("Fetch timeOut reached")
			return
		}
	}
}
