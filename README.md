daemon have been moved to [kenretto/daemon](https://github.com/kenretto/daemon) !

# daemon
golang start the daemon,
quickly build a golang service with its own daemon. Windows only support start command.

### Usage
The package support two method to run a go service
1. Directly let your service run away from the terminal session.
2. Run your service as a start stop restart command.

#### Note
The interface must be implement.
```go
type Worker interface {
	// PidSavePath pid file save path, such as /var/run
	PidSavePath() string
	// Name pid file name, like a, Then the complete PID file is called /var/run/a.pid
	Name() string
	// Start your service startup logic, like public static void main, do anything here, such as http.ListenAndServe
	Start()
	// Stop We must ensure that the running program exits gracefully instead of forcing an interrupt, 
    // So do some necessary actions before the service shuts down,
    // such as http.Shutdown(context.Background()) 
	Stop() error
	// Restart Similar to stop
	Restart() error
}
```

#### Example
```go
package main

import "github.com/medivh-jay/daemon"
```

- First, create an object implement daemon.Worker, such as ...

```go
package main

import (
    "context"
    "fmt"
    "github.com/medivh-jay/daemon"
    "net/http"
)

// HTTPServer example 
type HTTPServer struct {
	http *http.Server
}

// PidSavePath pid save path, save pid in workspace
func (httpServer *HTTPServer) PidSavePath() string {
	return "./"
}

// Name pid filename , pid file name is http
func (httpServer *HTTPServer) Name() string {
	return "http"
}

// Start start http server 
func (httpServer *HTTPServer) Start() {
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		fmt.Println("hello world")
		_, _ = writer.Write([]byte("hello world"))
	})
	httpServer.http = &http.Server{Handler: http.DefaultServeMux, Addr: ":9047"}
	_ = httpServer.http.ListenAndServe()
}

// Stop before stop http server operation, usually gracefully shut down the server
func (httpServer *HTTPServer) Stop() error {
	fmt.Println("Prepare to shut down the server")
	err := httpServer.http.Shutdown(context.Background())
	fmt.Println("The server has been shut down")
	return err
}

// Restart restart http server, usually gracefully shut down the server and then execute start
func (httpServer *HTTPServer) Restart() error {
	fmt.Println("The server is shutting down")
	err := httpServer.Stop()
	return err
}
```

- And then, create your main func

```go
package main

import (
    "github.com/medivh-jay/daemon"
    "log"
    "os"
)

func main() {
	// Custom output file, in fact, if you don't need the contents of the program's standard output and standard error output, you don't need this.
	out, _ := os.OpenFile("./http.log", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	err, _ := os.OpenFile("./http_err.log", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)

	// Use daemon.NewProcess to make your worker have signal monitoring, restart listening, and turn off listening, SetPipeline it's not necessary.
	proc := daemon.NewProcess(new(HTTPServer)).SetPipeline(nil, out, err)

	// This line is an example of creating a multi-level command
	daemon.GetCommand().AddWorker(proc).AddWorker(proc)
	
	// This line is an example of registering the main service directly
	daemon.Register(proc)

	// Start
	if rs := daemon.Run(); rs != nil {
		log.Fatalln(rs)
	}
}
```

- Of course, you can use the return object of daemon.NewProcess to let the service listen for the signal

```go
proc.On(syscall.SIGTERM, func() {
    fmt.Println("a custom signal")
})
```


```bash
go build -o myapp main.go
./myapp --help
./myapp start
./myapp restart
./myapp stop
```

#### Another

If you don't want to import spf13/cobra, Can be used directly 
```go
    var process = daemon.NewProcess(new(HTTPServer))
    _ = process.Run()
```
let the service run to the background

- You can use the GetCommand method to get the cobra.Command object to set up more command content
- command start have a flag --daemon

If you don't need the program to run as daemon mode for the time being,for example, you're using GoLand for debugging. You can set Program arguments to *(your app) start --daemon=false on Run/Debug Configurations of GoLand
