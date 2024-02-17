package main

import (
	dependency_tree "Repository-Scrapper/dependency-tree"
	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()
	dependencyTree := dependency_tree.NewDependencyTree()
	router.POST("/get-dependency-tree", dependencyTree.Get)
	router.Run(":8080")
}
