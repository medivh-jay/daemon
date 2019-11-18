package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
)

const (
	// EnvName 标识是子进程的环境变量名称
	EnvName = "DAEMON"
)

// Worker 具体工作程序接口
type Worker interface {
	// PidSavePath pid 文件保存路径
	PidSavePath() string
	// Name pid文件的名字
	Name() string
	// Start 启动服务的具体执行
	Start()
	// Stop 关闭服务的前置操作
	Stop() error
	// Restart 重启服务的前置操作
	Restart() error
}

type (
	// 系统信号处理方法
	signalHandlers map[os.Signal]func()
	// Process 启动服务的具体配置信息
	Process struct {
		Pipeline       [3]*os.File    // 输入输出管道, 0->input, 1->output, 2->err
		Pid            *Pid           // pid 对象信息
		worker         Worker         // 工作对象
		DaemonTag      string         // 标识是子进程的环境变量名称
		SignalHandlers signalHandlers // 信号处理器
	}
)

// Listen 监听系统信号
func (handlers signalHandlers) Listen() {
	var sig = make(chan os.Signal)
	signal.Notify(sig)
	for {
		received := <-sig
		if handler, ok := handlers[received]; ok {
			handler()
		}
	}
}

// NewProcess 实例化工作进程配置
//  worker 具体工作对象
func NewProcess(worker Worker) *Process {
	process := &Process{
		Pipeline: [3]*os.File{os.Stdin, os.Stdout, os.Stderr},
		Pid: &Pid{
			ServicesName: worker.Name(),
			SavePath:     worker.PidSavePath(),
			Pid:          os.Getpid(),
		},
		worker:    worker,
		DaemonTag: EnvName,
	}
	process.registerDefaultStopHandle()
	process.registerDefaultRestartHandle()
	return process
}

// SetPipeline 设置输入输出文件描述对象, 最多三个, 顺序分别为 0 -> 输入(一般直接放弃, 可以传nil), 1 -> 正常输出, 2 -> 错误输出
//  如果不需要程序运行时对标准输入输出的信息, 可以不用设置
func (process *Process) SetPipeline(pipes ...*os.File) *Process {
	if len(pipes) > 3 {
		pipes = pipes[0:3]
	}
	for index, pipe := range pipes {
		process.Pipeline[index] = pipe
	}
	return process
}

// SetDaemonTag 设置Daemon标识环境变量, 因为 golang 的进程机制无法像常规的进程管理一下正常fork,
// 所以使用的 exec.Command 启动一个进程, 这个方法启动的进程有一个问题是, 启动起来的子进程的 ppid 为当前用户 systemd,
// 所以以前的常用代码  fork 的 ppid == 1 这类操作在这里很难实现, 所以需要一个标识让 exec.Command 启动的本身知道自己是被 exec.Command 启动起来的,
// 有很多中方法, 但是大多都会侵入程序本身逻辑, 就没必要了, 所以使用一个环境变量来让子进程读取, 但是再怎么避免, 可能默认的环境变量名 DAEMON 都可能是具体程序
// 需要的环境变量, 所以这里提供方法修改默认的环境变量名称
func (process *Process) SetDaemonTag(name string) *Process {
	process.DaemonTag = name
	return process
}

// On 注册自定义的子进程的信号处理方法, 这里注册的方法实际是在子进程运行的, 子进程运行的程序逻辑是
// 真正的程序逻辑在子进程的一个协程中运行, 而子进程的主程序运行的信号监听方法
func (process *Process) On(signal os.Signal, fn func()) {
	if process.SignalHandlers == nil {
		process.SignalHandlers = make(signalHandlers)
	}
	process.SignalHandlers[signal] = fn
}

// 注册默认的关闭方法, 监听了 USR1 信号
func (process *Process) registerDefaultStopHandle() {
	process.On(SIGUSR1, func() {
		err := process.worker.Stop()
		if err != nil {
			_, _ = process.Pipeline[1].WriteString(err.Error())
		}
		process.Pid.Remove()
		os.Exit(0)
	})
}

// 注册默认的重启方法, 监听了 USR2 信号
func (process *Process) registerDefaultRestartHandle() {
	process.On(SIGUSR2, func() {
		var done = make(chan bool)
		go func() {
			err := process.worker.Restart()
			if err != nil {
				_, _ = process.Pipeline[1].WriteString(err.Error())
			}
			process.Pid.Remove()
			done <- true
		}()
		_ = os.Unsetenv(process.DaemonTag)
		err := process.Run()
		if err != nil {
			_, _ = process.Pipeline[1].WriteString(err.Error())
		}
		<-done
		os.Exit(0)
	})
}

// IsChild 判断是否是在子进程中启动的, 根据环境变量 DAEMON 判断
func (process *Process) IsChild() bool {
	return os.Getenv(process.DaemonTag) == "true"
}

// Run 运行程序,主逻辑在协程中运行,主协程运行系统信号监听程序
func (process *Process) Run() error {
	if process.IsChild() {
		if err := process.Pid.Save(); err != nil {
			return err
		}
		go process.worker.Start()
		process.SignalHandlers.Listen()
		return nil
	}

	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("%s=true", EnvName))
	cmd.Stdin, cmd.Stdout, cmd.Stderr = process.Pipeline[0], process.Pipeline[1], process.Pipeline[2]

	err := cmd.Start()
	if err != nil {
		return err
	}
	return cmd.Process.Release()

}
