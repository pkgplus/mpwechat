package mpwechat

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

const (
	MP_WECHAT_URL        = "https://mp.weixin.qq.com"
	REG_STR_NEWMSGNUM    = `(?s)class="number">(\d+)<\/em>[\r\n]+[^\r\n]+新消息`
	REG_STR_NEWADDFANS   = `(?s)class="number">(\d+)<\/em>[\r\n]+[^\r\n]+新增人数`
	REG_STR_NEWTOTALFANS = `(?s)class="number">(\d+)<\/em>[\r\n]+[^\r\n]+总用户数`
	REG_STR_TICKET       = `(?s)\Wticket:"(\S+)",`
	REG_STR_USERNAME     = `(?s)\Wuser_name:"(\S+)",`
	REG_STR_NICKNAME     = `(?s)\Wnick_name:"(\S+)",`
	REG_STR_UIN          = `(?s)\Wuin:"(\S+)"`
	REG_STR_UINBASE64    = `(?s)\Wuin_base64:"(\S+)"`
	REG_STR_LASTMSGID    = `(?s)class="message_item " id="msgListItem\d+" data-id="(\d+)">`
)

var REG_NEWMSGNUM, REG_NEWADDFANS, REG_TOTALFANS,
	REG_TICKET, REG_USERNAME, REG_NICKNAME,
	REG_UIN, REG_UINBASE64,
	REG_LASTMSGID *regexp.Regexp

func init() {
	REG_NEWMSGNUM = regexp.MustCompile(REG_STR_NEWADDFANS)
	REG_NEWADDFANS = regexp.MustCompile(REG_STR_NEWADDFANS)
	REG_TOTALFANS = regexp.MustCompile(REG_STR_NEWTOTALFANS)
	REG_TICKET = regexp.MustCompile(REG_STR_TICKET)
	REG_USERNAME = regexp.MustCompile(REG_STR_USERNAME)
	REG_NICKNAME = regexp.MustCompile(REG_STR_NICKNAME)
	REG_LASTMSGID = regexp.MustCompile(REG_STR_LASTMSGID)
	REG_UIN = regexp.MustCompile(REG_STR_UIN)
	REG_UINBASE64 = regexp.MustCompile(REG_STR_UINBASE64)
}

type WechatResponse struct {
	BaseResp struct {
		Ret    int    `json:"ret"`
		RetMsg string `json:"err_msg"`
	} `json:"base_resp"`
	RedirectUrl string `json:"redirect_url"`
}

type MpWechat struct {
	Email           string
	Pwd             string
	ImgCode         string
	F               string
	Token           string
	CookieFile      string
	NewMsgNum       int
	NewFansNum      int
	TotalFansNum    int
	Uin             string `json:"uin"`
	UinBase64       string `json:"uin_base64"`
	UserName        string `json:"user_name"`
	NickName        string `json:"nick_name"`
	Ticket          string `json:"ticket"`
	ChanMpSendMsg   chan *SendMessage
	ChanMpSendResp  chan *WXBaseResp
	ChanUserRecvMsg chan *RecvMessage
	ChanUserSendMsg chan *RecvMessage

	PrepareSendFlag bool
}

func (m *MpWechat) LoginMpWechat() error {
	v := url.Values{}
	v.Set("username", m.Email)
	v.Set("pwd", m.Pwd)
	v.Set("imgcode", m.ImgCode)
	v.Set("f", m.F)

	//New POST Request
	post_url := MP_WECHAT_URL + "/cgi-bin/login"
	req, req_err := http.NewRequest("POST", post_url, strings.NewReader(v.Encode()))
	if req_err != nil {
		return req_err
	}

	//add header
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Origin", MP_WECHAT_URL)
	req.Header.Add("Referer", MP_WECHAT_URL)

	//New HTTP CLIENT
	client := &http.Client{}
	resp, client_err := client.Do(req)
	if client_err != nil {
		return client_err
	}
	defer resp.Body.Close()

	//login result
	resp_body, _ := ioutil.ReadAll(resp.Body)

	//parse login result
	wechat_resp := &WechatResponse{}
	json_err := json.Unmarshal(resp_body, wechat_resp)
	if json_err != nil {
		return json_err
	}

	//保存cookie
	cookie_err := m.SaveCookies(resp.Cookies())
	if cookie_err != nil {
		return cookie_err
	}

	//跳转到主页(带cookie)
	url_home := MP_WECHAT_URL + wechat_resp.RedirectUrl
	req_home, req_home_err := http.NewRequest("GET", url_home, nil)
	if req_home_err != nil {
		return req_home_err
	}
	for _, c := range resp.Cookies() {
		req_home.AddCookie(c)
	}
	url_h, url_err := url.Parse(url_home)
	if url_err != nil {
		return url_err
	}
	m.Token = url_h.Query().Get("token")

	resp, client_err = client.Do(req_home)
	if client_err != nil {
		return client_err
	}

	resp_body, _ = ioutil.ReadAll(resp.Body)
	match_flag := m.GetLoginRet(resp_body)
	if !match_flag {
		return errors.New("登陆失败！")
	} else {
		m.ModifyCookies(resp.Cookies())
	}

	return nil
}

func (m *MpWechat) GetLoginRet(b []byte) bool {
	var matched_strs []string
	match_flag := false

	//新消息
	matched_strs = REG_NEWMSGNUM.FindStringSubmatch(string(b))
	if len(matched_strs) == 2 {
		m.NewMsgNum, _ = strconv.Atoi(matched_strs[1])
		match_flag = true
	}

	//新增人数
	matched_strs = REG_NEWADDFANS.FindStringSubmatch(string(b))
	if len(matched_strs) == 2 {
		m.NewFansNum, _ = strconv.Atoi(matched_strs[1])
		match_flag = true
	}

	//总用户数
	matched_strs = REG_TOTALFANS.FindStringSubmatch(string(b))
	if len(matched_strs) == 2 {
		m.TotalFansNum, _ = strconv.Atoi(matched_strs[1])
		match_flag = true
	}

	//ticket
	matched_strs = REG_TICKET.FindStringSubmatch(string(b))
	if len(matched_strs) == 2 {
		m.Ticket = matched_strs[1]
	}

	//user_name
	matched_strs = REG_USERNAME.FindStringSubmatch(string(b))
	if len(matched_strs) == 2 {
		m.UserName = matched_strs[1]
	}

	//nick_name
	matched_strs = REG_NICKNAME.FindStringSubmatch(string(b))
	if len(matched_strs) == 2 {
		m.NickName = matched_strs[1]
	}

	//uin or fakeid
	matched_strs = REG_UIN.FindStringSubmatch(string(b))
	if len(matched_strs) == 2 {
		m.Uin = matched_strs[1]
	}

	//uinbase64
	matched_strs = REG_UINBASE64.FindStringSubmatch(string(b))
	if len(matched_strs) == 2 {
		m.UinBase64 = matched_strs[1]
	}

	return match_flag
}
