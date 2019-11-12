# daemon
golang start the daemon,
quickly build a golang program with its own daemon

### examples
```go
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
```

### The method you need to use
- create a new process
```go
services := daemon.NewProcess("services", startWebServer)
```

- set OnStop event operation
```go
	services.OnStop = func() {
		log.Println("close server")
		ctx := context.Background()
		_ = server.Shutdown(ctx)
	}
```

- set OnRestart event operation
```go
	services.OnRestart = func() {
		ctx := context.Background()
		_ = server.Shutdown(ctx)
	}
```

- Monitoring semaphore
```go
	services.On(syscall.SIGTERM, func() {
		log.Println("a signal")
	})
```

- command stop and restart used signal USR1 and USR2

- register your process
```go
daemon.Register(services)
```

- Run
```go
daemon.Run()
```


```bash
go build -o myapp main.go
# default *.pid file save to /var/run/{myapp}*.pid , so , execute with sudo
sudo ./myapp services start
sudo ./myapp services restart
sudo ./myapp services stop
```
