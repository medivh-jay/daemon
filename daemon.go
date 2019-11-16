package daemon

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	command = &Daemon{command: &cobra.Command{Use: Name()}}
)

// Command 给自己的运行worker设置命令, 毕竟自己的程序也会需要各种参数, 如果实现了这个接口
// ,SetCommand 将在启动前执行, 传入 cobra.Command 对象, 可保存以供使用
type Command interface {
	SetCommand(cmd *cobra.Command)
}

func start(worker *Process) *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: fmt.Sprintf("start %s", worker.worker.Name()),
		Run: func(cmd *cobra.Command, args []string) {
			err := worker.Run()
			if err != nil {
				panic(err)
			}
		},
	}
}

func stop(worker *Process) *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: fmt.Sprintf("stop %s", worker.worker.Name()),
		Run: func(cmd *cobra.Command, args []string) {
			data, err := ioutil.ReadFile(worker.Pid.SaveFilename())
			if err != nil {
				panic(err)
			}
			pid, err := strconv.Atoi(string(data))
			if err != nil {
				panic(err)
			}
			process, err := os.FindProcess(pid)
			if err != nil {
				panic(err)
			}
			_ = process.Signal(syscall.SIGUSR1)
		},
	}
}

func restart(worker *Process) *cobra.Command {
	return &cobra.Command{
		Use:   "restart",
		Short: fmt.Sprintf("restart %s", worker.worker.Name()),
		Run: func(cmd *cobra.Command, args []string) {
			data, err := ioutil.ReadFile(worker.Pid.SaveFilename())
			if err != nil {
				panic(err)
			}
			pid, err := strconv.Atoi(string(data))
			if err != nil {
				panic(err)
			}
			process, err := os.FindProcess(pid)
			if err != nil {
				panic(err)
			}
			_ = process.Signal(syscall.SIGUSR2)
		},
	}
}

// Daemon 命令管理
type Daemon struct {
	command  *cobra.Command
	children map[string]*Daemon
	parent   *Daemon
	worker   *Process
}

// AddWorker 添加子执行程序
// 可链式调用生成多级命令
// 非链式调用生成多个同级的命令,但是记住同级的命令不要同名
func (daemon *Daemon) AddWorker(worker *Process) *Daemon {
	if daemon.children == nil {
		daemon.children = make(map[string]*Daemon)
	}

	child := &Daemon{command: &cobra.Command{Use: worker.worker.Name()}, parent: daemon}
	if _, ok := worker.worker.(Command); ok {
		worker.worker.(Command).SetCommand(child.command)
	}
	child.command.AddCommand(start(worker), stop(worker), restart(worker))
	daemon.command.AddCommand(child.command)
	daemon.children[worker.worker.Name()] = child
	return child
}

// GetParent 获取父级命令
func (daemon *Daemon) GetParent() *Daemon {
	return daemon.parent
}

// Register 注册主执行程序, 没有可不注册
func Register(worker *Process) {
	command.parent = nil
	command.worker = worker
	if _, ok := worker.worker.(Command); ok {
		worker.worker.(Command).SetCommand(command.command)
	}
	command.command.AddCommand(start(worker), stop(worker), restart(worker))
}

// GetCommand 获取主命令管理
func GetCommand() *Daemon {
	return command
}

// Run 运行入口
func Run() error {
	return command.command.Execute()
}

// Name 获取运行程序名称
func Name() string {
	fileInfo, err := os.Stat(os.Args[0])
	if err != nil {
		return ""
	}
	return fileInfo.Name()
}
