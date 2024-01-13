// 强密码正则表达式
var strongPasswd = /^(?=.*\d)(?=.*[a-z])(?=.*[A-Z]).{6,16}$/;
// 合法用户名正则表达式
var goodUsername = /^[\u4E00-\u9FA5A-Za-z0-9_]{2,20}$/;

// 检查密码强度及吻合性
function checkPasswd(){
	// 仅在HTML文档内搜索class中第一个元素
	const passwd = document.querySelector(".passPasswd");
	const confirm = document.querySelector(".passConfirm");
	const hidden = document.querySelector(".passMd5");
	// 使用正则检查密码是否合规
	if (strongPasswd.test(passwd.value))
	{
		// 检查二次输入是否吻合
		if (passwd.value === confirm.value)
		{
			// 锁定输入，同时禁止明文被提交
			passwd.disabled = true;
			confirm.disabled = true;
			// 加密并提交表单
			hidden.value = hex_md5(passwd.value);
			return true;
		}
		else{
			alert("密码不一致！");
		}
	}
	else
	{
		alert("密码不合规！");
	}
	passwd.disabled = false;
	confirm.disabled = false;
	return false;
}

// 提交登录信息
function checkLogin(){
	// 仅在HTML文档内搜索class中第一个元素
	const passwd = document.querySelector(".passPasswd");
	const hidden = document.querySelector(".passMd5");
	passwd.disabled = true;
	// 加密并提交表单
	hidden.value = hex_md5(passwd.value);
	return true;
}

// 检查注册信息
function checkReg(){
	// 仅在HTML文档内搜索class中第一个元素
	const name = document.querySelector(".passName");
	// 使用正则检查用户名是否合规
	if (name.value != "admin")
	{
		if (goodUsername.test(name.value))
		{
			return checkPasswd();
		}
		else
		{
			alert("用户名不合规！");
		}
	}
	else
	{
		alert("非法用户名！");
	}
	name.disabled = false;
	passwd.disabled = false;
	confirm.disabled = false;
	return false;
}