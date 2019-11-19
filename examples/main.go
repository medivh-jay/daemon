package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"syscall"

	"github.com/medivh-jay/daemon"
	"github.com/spf13/cobra"
)

// HTTPServer http 服务器示例
type HTTPServer struct {
	http *http.Server
	cmd  *cobra.Command
}

// PidSavePath pid保存路径
func (httpServer *HTTPServer) PidSavePath() string {
	return "./"
}

// Name pid文件名
func (httpServer *HTTPServer) Name() string {
	return "http"
}

// SetCommand 从 daemon 获得 cobra.Command 对象
func (httpServer *HTTPServer) SetCommand(cmd *cobra.Command) {
	// 在这里添加参数时他的参数不是对应服务的 start stop restart 命令的, 比如这个示例服务
	// 他对应的是示例服务命令, s所以这里添加的自定义 flag 应该在 start 之前传入
	cmd.PersistentFlags().StringP("test", "t", "yes", "")
	httpServer.cmd = cmd
}

// Start 启动web服务
func (httpServer *HTTPServer) Start() {
	fmt.Println(httpServer.cmd.Flags().GetString("test"))
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		fmt.Println("hello world")
		_, _ = writer.Write([]byte("hello world"))
	})
	httpServer.http = &http.Server{Handler: http.DefaultServeMux, Addr: ":9047"}
	_ = httpServer.http.ListenAndServe()
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
	proc.On(syscall.SIGTERM, func() {
		fmt.Println("a custom signal")
	})
	// 示例,多级命令服务
	// 这里的示例由于实现了 Command 接口, 所以这里会出现 flag test 不存在的情况, 实际情况, 每一个 worker 都应该是唯一的
	// 不要共享一个 worker 对象指针
	daemon.GetCommand().AddWorker(proc).AddWorker(proc)
	// 示例,主服务
	daemon.Register(proc)

	// 运行
	if rs := daemon.Run(); rs != nil {
		log.Fatalln(rs)
	}
}
