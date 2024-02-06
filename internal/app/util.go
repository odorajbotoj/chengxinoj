package app

import (
	"archive/zip"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

// 获取ip地址
func getIP(r *http.Request) string {
	forwarded := r.Header.Get("X-FORWARDED_FOR")
	if forwarded != "" {
		return strings.Split(forwarded, ":")[0]
	} else {
		return strings.Split(r.RemoteAddr, ":")[0]
	}
}

// 判断是否包含
func in(target string, str_array []string) bool {
	sort.Strings(str_array)
	index := sort.SearchStrings(str_array, target)
	if index < len(str_array) && str_array[index] == target {
		return true
	}
	return false
}

// 压缩文件夹
func zipFile(w io.Writer, srcs ...string) error {
	zw := zip.NewWriter(w)
	defer zw.Close()
	// 下面来将文件写入 zw ，因为有可能会有很多个目录及文件，所以递归处理
	for _, src := range srcs {
		err := filepath.WalkDir(src, func(path string, d fs.DirEntry, er error) error {
			if er != nil {
				return er
			}
			fi, er := d.Info()
			if er != nil {
				return er
			}
			fh, er := zip.FileInfoHeader(fi)
			if er != nil {
				return er
			}
			fh.Name = strings.TrimPrefix(path, string(filepath.Separator))
			if fi.IsDir() {
				fh.Name += string(filepath.Separator)
			}
			// 在zip里面新建文件
			w, er := zw.CreateHeader(fh)
			if er != nil {
				return er
			}
			if !fh.Mode().IsRegular() {
				return nil
			}
			// 打开待压缩的文件
			fr, er := os.Open(path)
			if er != nil {
				return er
			}
			defer fr.Close()
			// 复制
			_, er = io.Copy(w, fr)
			if er != nil {
				return er
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}
