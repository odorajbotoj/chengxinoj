# chengxinoj

澄心OJ - chengxinoj. A simple OJ.

+ Beta测试中……

+ Windows最低系统需求：Windows7 32位

+ **重要**：windows与macOS下judger**不工作**，此问题正在尝试解决中。**暂无其他系统测试信息**。

+ 项目定位：机房环境“微OJ”

## 构建

+ 需要go 1.20+

+ `go build`即可。可交叉编译，理论上多平台支持。

## 使用说明

### 管理员侧

#### 第一次启动

+ 会生成一些文件与目录。我们需要注意的是`config.json`，内部有三个配置项，分别是标题、端口、管理员密码散列。

+ 标题即页面左上角显示的文字。

+ 端口为服务开放的端口。数字前面要**加英文冒号** 。

+ 管理员密码散列为一串MD5字符串。英文字母必须**小写** 。

+ 修改配置后需要**重新启动服务** 。

#### 进入控制台

+ 输入`admin`和密码进行登陆。服务端每次重启后，管理员**必须**重新登陆。

#### 主页 用户

+ 注册用户：打开用户注册页面。管理员可**无视**注册禁令进行注册。

+ 管理用户：进入用户管理界面。

+ 启用/禁用注册：关闭或开启**注册禁令**。注册禁令生效时，用户无法注册。

#### 用户管理页

+ 导入用户：选择导出的`db`文件进行导入。

+ 导出选中：导出一个`db`文件。

+ 删除选中：顾名思义。

#### 主页 比赛

+ 导入比赛：选择导出的`zip`文件进行导入。

+ 导出比赛：导出一个`zip`文件

+ 开始/结束比赛：顾名思义。开始比赛后，管理员不能进行打包等操作，以免收到脏数据。比赛结束或未开始时，用户无法下载或提交文件。

#### 主页 下发文件

+ 上传的文件将供学生下载。比赛开始后管理员将不能上传或删除文件，学生只能下载下发的文件。

#### 任务点

+ 名称：只能输入纯英文。

+ 删除选中：顾名思义。

+ 打包下载：下载全部学生提交的文件。导出一个`zip`文件。

+ 清空上传：清空全部学生提交的文件及记录。**建议在导入比赛后执行以清理工作区**。

+ **管理员可以提交答案，但不会出现在榜单里。**

#### 编辑任务点

+ 网页内均注明。

#### 榜单

+ 用户以AC数量从多到少排序。AC数相同时以用户名字典序排序。

+ 点击相应按钮可以查看测试点详情。

### 用户侧

#### 比赛

+ 比赛未开始时，请耐心等待。

+ 比赛开始后，可以下载教师下发的文件，可以提交自己的作业。

+ **建议在开始时执行`清空上传`以清理工作区。**

## 第三方

+ `md5.js` [Version 2.2 Copyright (C) Paul Johnston 1999 - 2009 | Distributed under the BSD License](http://pajhome.org.uk/crypt/md5)

+ `github.com/tidwall/buntdb` [License: MIT](https://github.com/tidwall/buntdb) 
