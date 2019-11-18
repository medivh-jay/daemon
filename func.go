package daemon

import (
	"os"
	"path"
)

// 锁文件
func lock(file *os.File) error {
	err := Flock(int(file.Fd()), LOCK_EX|LOCK_NB)
	return err
}

// 写文件
func write(filename string, body string) (file *os.File, err error) {
	file, err = create(filename)
	if err != nil {
		return
	}

	_, err = file.WriteString(body)
	return
}

// 创建文件
func create(filename string) (file *os.File, err error) {
	dir := path.Dir(filename)
	_, err = os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(dir, 0644)
			if err != nil {
				return
			}
		} else {
			return
		}
	}
	file, err = os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0644)
	if err = lock(file); err != nil {
		return
	}
	return
}
