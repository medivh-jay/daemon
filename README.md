# daemon
golang start the daemon,
quickly build a golang service with its own daemon. Windows only support start command

### Usage

```go
import "github.com/medivh-jay/daemon"
```

- First, create a object implement daemon.Worker, like this

```go
// HTTPServer example 
type HTTPServer struct {
	http *http.Server
}

// PidSavePath pid save path
func (httpServer *HTTPServer) PidSavePath() string {
	return "./"
}

// Name pid filename , and its command name
func (httpServer *HTTPServer) Name() string {
	return "http"
}

// Start start http server 
func (httpServer *HTTPServer) Start() {
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		fmt.Println("hello world")
		writer.Write([]byte("hello world"))
	})
	httpServer.http = &http.Server{Handler: http.DefaultServeMux, Addr: ":9047"}
	httpServer.http.ListenAndServe()
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

- You can use the GetCommand method to get the cobra.Command object to set up more command content
