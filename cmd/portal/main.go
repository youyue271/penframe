package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"penframe/internal/portal"
)

func main() {
	listenAddr := flag.String("listen", ":8080", "监听地址")
	toolsPath := flag.String("tools", "examples/mvp/tools.yaml", "工具目录配置路径")
	workflowPath := flag.String("workflow", "examples/mvp/workflow.yaml", "工作流配置路径")
	flag.Parse()

	server, err := portal.NewServer(*toolsPath, *workflowPath)
	if err != nil {
		log.Fatalf("启动控制台失败：%v", err)
	}

	fmt.Printf("控制台已启动：http://localhost%s\n", *listenAddr)
	if err := http.ListenAndServe(*listenAddr, server); err != nil {
		log.Fatalf("监听失败：%v", err)
	}
}
