package main

import (
	"coredemo/framework"
	"net/http"
)

func main() {
	// 创建 Server
	server := &http.Server{
		// 自定义的请求核心处理函数
		Handler: framework.NewCore(),
		// 请求监听端口
		Addr:    ":8080",
	}
	server.ListenAndServe()
}
