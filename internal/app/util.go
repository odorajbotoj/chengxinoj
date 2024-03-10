package app

import (
	"archive/zip"
	"fmt"
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
	// 下面来将文件写入 zw
	for _, src := range srcs {
		src = strings.TrimSuffix(src, "/")
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
			fh.Name = filepath.ToSlash(path)
			if fi.IsDir() {
				fh.Name += "/"
			} else {
				// 设置zip的文件压缩算法
				fh.Method = zip.Deflate
			}
			// 在zip里面新建文件
			w, er := zw.CreateHeader(fh)
			if er != nil {
				return er
			}
			if !fh.Mode().IsRegular() || fi.IsDir() {
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

// 解压缩
func unzipFile(zipf io.ReaderAt, size int64, dst string) error {
	zipr, err := zip.NewReader(zipf, size)
	if err != nil {
		return err
	}
	for _, f := range zipr.File {
		filePath := f.Name
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(filePath, 0755)
			continue
		}
		// 创建对应文件夹
		err = os.MkdirAll(filepath.Dir(filePath), 0755)
		if err != nil {
			return err
		}
		// 解压到的目标文件
		err = os.RemoveAll(dst + filePath)
		if err != nil {
			return err
		}
		dstFile, err := os.OpenFile(dst+filePath, os.O_WRONLY|os.O_CREATE, f.Mode())
		if err != nil {
			return err
		}
		file, err := f.Open()
		if err != nil {
			return err
		}
		// 写入到解压到的目标文件
		if _, err = io.Copy(dstFile, file); err != nil {
			return err
		}
		dstFile.Close()
		file.Close()
	}
	return nil
}

func copyFile(src string, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	_, err = io.Copy(destination, source)

	source.Close()
	destination.Close()
	return err
}

func alertAndRedir(w http.ResponseWriter, alert string, redir string) {
	w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("` + alert + `");window.location.replace("` + redir + `");</script>`))
}
