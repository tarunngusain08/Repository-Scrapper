package dependency_tree

import (
	"bufio"
	"fmt"
	"log"
	"strings"
	"time"
)

func (d *DependencyTree) parseGoMod(goModContent, parentUrl string) error {

	scanner := bufio.NewScanner(strings.NewReader(goModContent))
	parent := d.RepositoryToArtifactMap[parentUrl]
	flag := false
	var name string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) > 0 && line[0] == ')' {
			flag = false
		}
		if strings.HasPrefix(line, "require") || flag {
			parts := strings.Fields(line)
			if len(parts) == 0 || len(parts) != 3 && !flag {
				flag = true
				continue
			}

			var version string
			if flag {
				name = parts[0]
				version = parts[1]
			} else {
				name = parts[1]
				version = parts[2]
			}

			// Check if the dependency already exists in the map
			if artifact, ok := d.RepositoryToArtifactMap[name]; ok {
				d.mu.Lock()
				parent.Dependencies = append(parent.Dependencies, artifact)
				d.mu.Unlock()
				continue
			}

			// Create a new artifact for the dependency
			artifact := &Artifact{
				Name:         name,
				Version:      version,
				Dependencies: make([]*Artifact, 0),
			}

			// Update the repository map and dependencies
			d.mu.Lock()
			d.RepositoryToArtifactMap[name] = artifact
			parent.Dependencies = append(parent.Dependencies, artifact)
			d.mu.Unlock()

			if name != "" {
				d.RepositoryChannel <- name
				//d.mu.Lock()
				//d.LastTimeDataSentToRepositoryChannel = time.Now()
				//d.mu.Unlock()
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error scanning go.mod content: %v", err)
	}
	return nil
}

func (d *DependencyTree) parse() {
	defer func() {
		log.Println("returning from parse")
		d.wg.Done()
		close(d.RepositoryChannel)
	}()

	timeout := time.NewTimer(5 * time.Second) // Set initial timeout
	for {
		select {
		case goMod, ok := <-d.GoModChanel:
			//d.mu.Lock()
			//timeDiff := time.Now().Sub(d.LastTimeDataSentToGoModChannel)
			//d.mu.Unlock()
			if !ok {
				return // Channel closed, no more data will be sent
			}
			d.wg.Add(1)
			go func(goModContent, parentUrl string) {
				defer d.wg.Done()
				err := d.parseGoMod(goModContent, parentUrl)
				if err != nil {
					d.ErrorChannel <- err
					return
				}
				if !timeout.Stop() {
					<-timeout.C
				}
				timeout.Reset(5 * time.Second)
			}(goMod.GoModContent, goMod.ParentUrl)
		case <-timeout.C:
			log.Println("Parse timeout reached")
			return
		}
	}
}
