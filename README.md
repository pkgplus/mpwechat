#mpwechat

##登陆微信公众平台
```go
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
```

## 获取用户信息(第一页)
```go
//获取关注用户信息(第一页)
wechatFans, fan_err := mp_wechat.GetFans()
if fan_err != nil {
    fmt.Println(fan_err)
    return
}
```

## 发送消息
### 发送消息前初始化
```go
mp_wechat.PrepareMpSendMsg()
```

### 发送文本
- `fan`类型为WechatFan
- `SendTextBlock`阻塞方式发送，等待发送结果
```
content := "你好！"
fmt.Printf("send %s to %v\n", content, fan.ID)
send_err := mp_wechat.SendTextBlock(fan.ID, content, 10)
if send_err != nil {
    fmt.Println(send_err)
}
```

### 发送图片
`SendImageBlock`阻塞方式发送，等待发送结果
>上传图片耗时长，建议使用非阻塞方法 SendImage

```go
file_name := `C:\Users\xuebing\Desktop\a.jpg`
fmt.Printf("send %s to %v\n", file_name, fan.ID)
send_err = mp_wechat.SendImageBlock(fan.ID, file_name, 30)
if send_err != nil {
    fmt.Println(send_err)
}
```

### 接收用户发送消息
> 被动接收用户消息，建议使用官方开放API

```go
resp_msg, err := mp_wechat.WaitUserSendMsg(&fan, 100)
if err != nil {
    fmt.Println(err)
} else {
    fmt.Printf("收到用户(%s)回复：%s\n", fan.NickName, resp_msg.Content)
}
```
