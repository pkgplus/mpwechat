package main

import (
	"fmt"
	"github.com/xuebing1110/mpwechat"
	_ "time"
)

func main() {
	//登陆
	mp_wechat := &mpwechat.MpWechat{
		Email:      "****@163.com",
		Pwd:        "md5",
		F:          "json",
		CookieFile: `F://mpwechat.qq.cookie`}
	login_err := mp_wechat.LoginMpWechat()
	if login_err != nil {
		fmt.Printf("登陆失败：%v", login_err)
	} else {
		fmt.Printf("新消息:%d,新增人数:%d,总人数:%d\n",
			mp_wechat.NewMsgNum,
			mp_wechat.NewFansNum,
			mp_wechat.TotalFansNum)
	}

	//获取关注用户信息(第一页)
	wechatFans, fan_err := mp_wechat.GetFans()
	if fan_err != nil {
		fmt.Println(fan_err)
		return
	}

	//给用户昵称为“***”的用户发送 文本及图片消息
	for _, fan := range wechatFans.Contacts {
		if fan.NickName != "***" {
			continue
		}
		mp_wechat.PrepareMpSendMsg()

		content := "你好！"
		fmt.Printf("send %s to %v\n", content, fan.ID)
		send_err := mp_wechat.SendTextBlock(fan.ID, content, 10)
		if send_err != nil {
			fmt.Println(send_err)
		}

		file_name := `C:\Users\xuebing\Desktop\a.jpg`
		fmt.Printf("send %s to %v\n", file_name, fan.ID)
		send_err = mp_wechat.SendImageBlock(fan.ID, file_name, 30)
		if send_err != nil {
			fmt.Println(send_err)
		}

		//获取用户回复(次功能完全可以用微信开放API)
		/*resp_msg, err := mp_wechat.WaitUserSendMsg(&fan, 100)
		  if err != nil {
		      fmt.Println(err)
		  } else {
		      fmt.Printf("收到用户(%s)回复：%s\n", fan.NickName, resp_msg.Content)
		  }*/
		break
	}
	//fmt.Println(wechatFans)
}
