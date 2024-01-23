package app

import (
	"os"
)

// 判断文件或目录是否存在
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// 判断文件是否存在，若不存在则创建
func checkFile(path string) error {
	exi, err := exists(path)
	if err != nil {
		return err
	}
	if !exi {
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		defer f.Close()
	}
	return nil
}

// 判断目录是否存在，若不存在则创建
func checkDir(path string) error {
	exi, err := exists(path)
	if err != nil {
		return err
	}
	if !exi {
		err = os.MkdirAll(path, 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

// 获取文件列表
func getFileList(path string) map[string]int64 {
	var ret = make(map[string]int64)
	rd, err := os.ReadDir(path)
	if err != nil {
		elog.Println("getSendList: ", err)
		return ret
	}
	for _, fi := range rd {
		if !fi.IsDir() {
			info, err := fi.Info()
			if err != nil {
				continue
			}
			ret[info.Name()] = info.Size()
		}
	}
	return ret
}
