package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	"penframe/internal/portal"
)

func main() {
	listenAddr := flag.String("listen", ":8080", "listen address")
	toolsPath := flag.String("tools", "", "tool catalog path (required)")
	workflowPath := flag.String("workflow", "", "workflow config path (required)")
	expURL := flag.String("exp-url", "", "Python exploit service URL (e.g. http://127.0.0.1:8787)")
	flag.Parse()
	if strings.TrimSpace(*toolsPath) == "" || strings.TrimSpace(*workflowPath) == "" {
		log.Fatal("both -tools and -workflow are required")
	}

	var server *portal.Server
	var err error
	if *expURL != "" {
		server, err = portal.NewServerWithExpURL(*toolsPath, *workflowPath, *expURL)
	} else {
		server, err = portal.NewServer(*toolsPath, *workflowPath)
	}
	if err != nil {
		log.Fatalf("failed to start portal: %v", err)
	}

	fmt.Printf("Portal started: http://localhost%s\n", *listenAddr)
	fmt.Println("Vue dev server: http://localhost:5173")
	if *expURL != "" {
		fmt.Printf("Exp service: %s\n", *expURL)
	}
	if err := http.ListenAndServe(*listenAddr, server); err != nil {
		log.Fatalf("listen failed: %v", err)
	}
}
