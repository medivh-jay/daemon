package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// Pid 这里边主要记录的进程id信息和进程pid文件描述符
type Pid struct {
	ServicesName string   // 服务名称,非进程名称
	SavePath     string   // 保存路径
	Pid          int      // 进程号
	File         *os.File // 文件句柄
}

// SaveFilename 获取保存pid的路径
func (pid Pid) SaveFilename() string {
	path, err := filepath.Abs(pid.SavePath)
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf("%s/%s.pid", path, pid.ServicesName)
}

// Save 保存 pid
func (pid Pid) Save() error {
	var err error
	pid.File, err = write(pid.SaveFilename(), strconv.Itoa(pid.Pid))
	return err
}

// Remove 关闭文件描述符并删除pid文件
func (pid Pid) Remove() {
	pid.File.Close()
	_ = os.Remove(pid.SaveFilename())
}
