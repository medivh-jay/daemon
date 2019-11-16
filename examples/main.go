package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/medivh-jay/daemon"
)

// HTTPServer http 服务器示例
type HTTPServer struct {
	http *http.Server
}

// PidSavePath pid保存路径
func (httpServer *HTTPServer) PidSavePath() string {
	return "./"
}

// Name pid文件名
func (httpServer *HTTPServer) Name() string {
	return "http"
}

// Start 启动web服务
func (httpServer *HTTPServer) Start() {
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		fmt.Println("hello world")
		writer.Write([]byte("hello world"))
	})
	httpServer.http = &http.Server{Handler: http.DefaultServeMux, Addr: ":9047"}
	httpServer.http.ListenAndServe()
}

// Stop 关闭web服务
func (httpServer *HTTPServer) Stop() error {
	fmt.Println("准备关闭服务器")
	err := httpServer.http.Shutdown(context.Background())
	fmt.Println("服务器已经关闭")
	return err
}

// Restart 重启web服务前关闭http服务
func (httpServer *HTTPServer) Restart() error {
	fmt.Println("服务器关闭中")
	err := httpServer.Stop()
	return err
}

func main() {
	// 自定义输出文件
	out, _ := os.OpenFile("./http.log", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	err, _ := os.OpenFile("./http_err.log", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)

	// 初始化一个新的运行程序
	proc := daemon.NewProcess(new(HTTPServer)).SetPipeline(nil, out, err)

	// 示例,多级命令服务
	daemon.GetCommand().AddWorker(proc).AddWorker(proc)
	// 示例,主服务
	daemon.Register(proc)

	// 运行
	if rs := daemon.Run(); rs != nil {
		log.Fatalln(rs)
	}
}
