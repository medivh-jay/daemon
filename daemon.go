// Package daemon golang daemon process
//  not support Windows
package daemon

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
)

var (
	// Stdin input pipe
	Stdin = os.Stdin
	// Stdout output pipe
	Stdout = os.Stdout
	// Stderr error pipe
	Stderr = os.Stderr
	// PidPath pid save path, default is /var/run/{app}/
	PidPath, _ = filepath.Abs("/var/run/" + Name())
)

var (
	// ShortDesc short desc for cobra.Command
	ShortDesc = `start a new daemon process`
	// LongDesc long desc for cobra.Command
	LongDesc = `start a new daemon process with os.StartProcess and cobra`

	// OperationStopShortDesc stop command short desc
	OperationStopShortDesc = `stop process`
	// OperationStopLongDesc stop command long desc
	OperationStopLongDesc = `Kill causes the Process to exit immediately. Kill does not wait until the Process has actually exited. 
This only kills the Process itself, not any other processes it may have started.`

	// OperationRestartShortDesc restart short desc
	OperationRestartShortDesc = `restart process`
	// OperationRestartLongDesc restart long desc
	OperationRestartLongDesc = `this command does not restart the service itself, instead, send a USR2 signal to the running process, 
restart logic is handled by the program itself`
)

var (
	// all process
	processes = make(map[string]*Process)
	// master app command
	command = &cobra.Command{Use: Name(), Short: ShortDesc, Long: LongDesc}

	// start command for every process
	start = &cobra.Command{
		Use:   "start",
		Short: "start process",
		Run: func(cmd *cobra.Command, args []string) {
			name := cmd.Parent().Name()
			attr := &os.ProcAttr{
				Env:   os.Environ(),
				Files: []*os.File{Stdin, Stdout, Stderr},
			}
			attr.Sys = &syscall.SysProcAttr{
				Setsid: true,
			}

			if os.Getppid() != 1 {
				// 父进程直接执行的代码,在这里启动子进程
				_, err := os.StartProcess(os.Args[0], os.Args, attr)
				if err != nil {
					log.Fatalln(err)
				}
				return
			}
			if isRunning(name) {
				log.Fatalln(errors.New("process is running"))
			}
			err := savePid(name, os.Getpid())
			if err != nil {
				log.Fatalln(err)
			}
			go func() {
				// 真正的程序逻辑在子进程的一个协程中运行
				processes[name].run(cmd, args)
			}()
			// 子进程的主程序监听所有信号量
			processes[name].signal = make(chan os.Signal)
			signal.Notify(processes[name].signal)
			for {
				received := <-processes[name].signal
				switch {
				case received == syscall.SIGUSR1:
					processes[name].OnStop()
					_ = os.Remove(fmt.Sprintf("%s/%s.pid", PidPath, name))
					return
				case received == syscall.SIGUSR2:
					var closed = make(chan int)
					go func() {
						processes[name].OnRestart()
						closed <- 1
					}()

					_ = os.Remove(fmt.Sprintf("%s/%s.pid", PidPath, name))
					os.Args[2] = "start"
					_, err = os.StartProcess(os.Args[0], os.Args, attr)
					if err != nil {
						log.Fatalln(err)
					}
					<-closed
					return
				default:
					operation, ok := processes[name].signalOperation[received]
					if ok {
						operation()
					}
				}
			}
		},
	}

	// stop command for every process
	stop = &cobra.Command{
		Use:   "stop",
		Short: OperationStopShortDesc,
		Long:  OperationStopLongDesc,
		Run: func(cmd *cobra.Command, args []string) {
			name := cmd.Parent().Name()
			process, err := os.FindProcess(findPid(name))
			if err != nil {
				log.Fatalln(err)
			}
			_ = process.Signal(syscall.SIGUSR1)
		},
	}

	// restart command for every process
	restart = &cobra.Command{
		Use:   "restart",
		Short: OperationRestartShortDesc,
		Long:  OperationRestartLongDesc,
		Run: func(cmd *cobra.Command, args []string) {
			name := cmd.Parent().Name()
			process, err := os.FindProcess(findPid(name))
			if err != nil {
				log.Fatalln(err)
			}
			_ = process.Signal(syscall.SIGUSR2)
		},
	}
)

// Process 具体程序
type Process struct {
	Name            string                                  // name for Register
	shortDesc       string                                  // process shot desc, can be null
	longDesc        string                                  // process long desc, can be null
	run             func(cmd *cobra.Command, args []string) // same as your main
	OnStop          func()                                  // stop command operation, will be executed when the signal USR1 is sent
	OnRestart       func()                                  // restart command operation, will be executed when the signal USR2 is sent
	SetCommand      func(cmd *cobra.Command)                // custom your process cobra.Command
	signal          chan os.Signal                          // pipe for os.Signal
	signalOperation map[os.Signal]func()                    // all registered signal
}

// NewProcess create a new process
func NewProcess(name string, run func(cmd *cobra.Command, args []string)) (process *Process) {
	process = &Process{
		Name:            name,
		run:             run,
		signalOperation: make(map[os.Signal]func()),
		OnStop: func() {
			_, _ = Stdout.WriteString(fmt.Sprintf("process [%s] will stop\n", name))
		},
		OnRestart: func() {
			_, _ = Stdout.WriteString(fmt.Sprintf("process [%s] will restart\n", name))
		},
	}
	return
}

// On custom operation for os.Signal
func (process *Process) On(signal syscall.Signal, fn func()) {
	process.signalOperation[signal] = fn
}

// Desc process short description and long description
func (process *Process) Desc(short, long string) *Process {
	process.shortDesc, process.longDesc = short, long
	return process
}

// Name get binary program name
func Name() string {
	stat, _ := os.Stat(os.Args[0])
	return stat.Name()
}

// Register register process
func Register(proc ...*Process) {
	for _, process := range proc {
		processes[process.Name] = process
		if process.Name == "" {
			panic("process name can not be null")
		}
		var cmd = &cobra.Command{
			Use: process.Name,
		}

		if process.shortDesc != "" {
			cmd.Short = process.shortDesc
		}
		if process.longDesc != "" {
			cmd.Long = process.longDesc
		}
		if process.SetCommand != nil {
			process.SetCommand(cmd)
		}

		cmd.AddCommand(start, stop, restart)
		command.AddCommand(cmd)
	}
}

// Run program execution entry, fn can be nil, if not nil,  then the master program will have Run functionality
func Run(fn ...func()) {
	if len(fn) == 1 {
		command.Run = func(cmd *cobra.Command, args []string) {
			fn[0]()
		}
	}
	err := command.Execute()
	if err != nil {
		panic(err)
	}
}

// Find find running process with pid
func Find(pid int) *os.Process {
	var process *os.Process
	process, _ = os.FindProcess(pid)
	return process
}

func isRunning(name string) bool {
	_, err := os.Stat(fmt.Sprintf("%s/%s.pid", PidPath, name))
	return !os.IsNotExist(err)
}

func findPid(name string) int {
	filename := fmt.Sprintf("%s/%s.pid", PidPath, name)
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalln(err)
	}

	pid, err := strconv.Atoi(string(body))
	if err != nil {
		log.Fatalln(err)
	}

	return pid
}

// 保存pid
func savePid(name string, pid int) (err error) {
	_, err = os.Stat(PidPath)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(PidPath, 0755)
			if err != nil {
				return
			}
		} else {
			return
		}
	}

	fd, err := os.Create(fmt.Sprintf("%s/%s.pid", PidPath, name))
	if err != nil {
		return
	}

	_, err = fd.WriteString(strconv.Itoa(pid))
	return
}
