package main

import (
	"context"
	"github.com/medivh-jay/daemon"
	"github.com/spf13/cobra"
	"log"
	"net/http"
	"os"
	"strconv"
	"syscall"
)

var server *http.Server

func startWebServer(cmd *cobra.Command, args []string) {
	log.Println("starting web server")
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte("hello world " + strconv.Itoa(os.Getpid())))
	})

	server = &http.Server{
		Addr:    ":9047",
		Handler: http.DefaultServeMux,
	}

	err := server.ListenAndServe()
	if err == http.ErrServerClosed {
		log.Println("server is closed")
		return
	}
	log.Println(err)
}

func main() {
	services := daemon.NewProcess("services", startWebServer)

	services.OnStop = func() {
		log.Println("close server")
		ctx := context.Background()
		_ = server.Shutdown(ctx)
	}

	services.OnRestart = func() {
		ctx := context.Background()
		_ = server.Shutdown(ctx)
	}

	services.On(syscall.SIGTERM, func() {
		log.Println("接收到了一个自定义的信号量")
	})

	daemon.Register(services)
	daemon.Run()
}
