package app

import (
	"archive/zip"
	"bytes"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

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
func zipFile(src string) ([]byte, error) {
	var b bytes.Buffer
	// 通过 b 来创建 zip.Write
	zw := zip.NewWriter(&b)
	defer zw.Close()
	// 下面来将文件写入 zw ，因为有可能会有很多个目录及文件，所以递归处理
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
			fh.Name += "/"
		}
		w, er := zw.CreateHeader(fh)
		if er != nil {
			return er
		}
		if !fh.Mode().IsRegular() {
			return nil
		}
		fr, er := os.Open(path)
		if er != nil {
			return er
		}
		defer fr.Close()
		_, er = io.Copy(w, fr)
		if er != nil {
			return er
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
