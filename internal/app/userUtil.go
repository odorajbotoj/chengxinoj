package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/tidwall/buntdb"
)

// 检查登录过期
func checkUser(r *http.Request) (UserData, bool) {
	var ud UserData // 用户信息
	ud.Name = ""
	ud.IsLogin = false
	ud.IsAdmin = false
	// 获取用户名
	c1, err := r.Cookie("username")
	if err != nil {
		if err != http.ErrNoCookie {
			elog.Println(err)
		}
		return ud, false
	}
	ud.Name, err = url.QueryUnescape(c1.Value)
	if err != nil {
		elog.Println(err)
		return ud, false
	}
	// 获取token
	c2, err := r.Cookie("token")
	if err != nil {
		if err != http.ErrNoCookie {
			elog.Println(err)
		}
		return ud, false
	}
	// 比对
	u, err := getUser(ud.Name)
	if err != nil {
		if err != buntdb.ErrNotFound {
			elog.Println(err)
		}
		return ud, false
	}
	if c2.Value == u.Token {
		ud.IsLogin = true
		if ud.Name == "admin" {
			ud.IsAdmin = true
		}
		return ud, false
	} else {
		return ud, true
	}
}

// 获取用户列表
func getUserList() []string {
	s := make([]string, 0)
	err := udb.View(func(tx *buntdb.Tx) error {
		e := tx.Ascend("name", func(key, value string) bool {
			var u User
			json.Unmarshal([]byte(value), &u)
			if u.Name != "admin" {
				s = append(s, u.Name)
			}
			return true // continue iteration
		})
		return e
	})
	if err != nil {
		elog.Println(err)
	}
	return s
}

// 获取用户信息
func getUser(name string) (User, error) {
	var err error
	var u User
	err = udb.View(func(tx *buntdb.Tx) error {
		var e error
		s, e := tx.Get("user:" + name + ":info")
		if e != nil {
			return e
		}
		e = json.Unmarshal([]byte(s), &u)
		return e
	})
	return u, err
}

// 写入用户信息
func setUser(name, md5 string) error {
	var err error
	// 写入用户数据
	var u User
	err = udb.Update(func(tx *buntdb.Tx) error {
		u.Name = name
		u.Md5 = md5
		b, e := json.Marshal(u)
		if e != nil {
			return e
		}
		_, _, e = tx.Set("user:"+name+":info", string(b), nil)
		if e != nil {
			return e
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

// 注册用户
func userReg(name, md5 string) error {
	// 过滤非法注册、注册admin
	if name == "admin" || !goodUserName.MatchString(name) {
		return fmt.Errorf("非法用户名")
	}
	// 数据操作
	err := udb.View(func(tx *buntdb.Tx) error {
		// 过滤重复注册
		_, e := tx.Get("user:" + name + ":info")
		return e
	})
	if err != nil {
		if err == buntdb.ErrNotFound {
			// 说明可以注册
			err = checkDir("recvFiles/" + name)
			if err != nil {
				return err
			}
			err = setUser(name, md5)
			if err != nil {
				return err
			}
			// success
			return nil
		} else {
			// 其他错误
			return err
		}
	} else {
		// 用户已存在
		return fmt.Errorf("用户已存在")
	}
}
