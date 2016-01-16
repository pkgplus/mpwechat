package mpwechat

import (
	"encoding/json"
	"errors"
	"fmt"
	//json_sample "github.com/bitly/go-simplejson"
	"io/ioutil"
	"net/http"
	"regexp"
)

const (
	REG_STR_GETFANS = `(?s)friendsList\s*:\s*\(({.*})\).contacts`
)

var REG_GETFANS *regexp.Regexp

func init() {
	REG_GETFANS = regexp.MustCompile(REG_STR_GETFANS)
}

type WechatFans struct {
	Contacts []WechatFan `json:"contacts"`
}

type WechatFan struct {
	ID          string       `json:"id"`
	NickName    string       `json:"nick_name"`
	RemarkName  string       `json:"remark_name"`
	GroupID     int          `json:"group_id"`
	HeadimgUrl  string       `json:"wx_headimg_url"`
	LastMsgInfo *LastMsgInfo `json:",omitempty"`
	WatchedMsgs bool         `json:",omitempty"`
}

type LastMsgInfo struct {
	LastMsgID         string
	LastMsgFromFakeID string
	CreateTime        string
}

func (fan *WechatFan) SetLastMsgInfo(recv_msg *RecvMessage) {
	if fan.LastMsgInfo == nil {
		fan.LastMsgInfo = &LastMsgInfo{
			fmt.Sprintf("%v", recv_msg.ID),
			recv_msg.FakeID,
			fmt.Sprintf("%v", recv_msg.DateTime)}
	} else {
		fan.LastMsgInfo.LastMsgID = fmt.Sprintf("%v", recv_msg.ID)
		fan.LastMsgInfo.LastMsgFromFakeID = recv_msg.FakeID
		fan.LastMsgInfo.CreateTime = fmt.Sprintf("%v", recv_msg.DateTime)
	}
}

func (m *MpWechat) GetFans() (*WechatFans, error) {
	wfans := &WechatFans{}
	url := fmt.Sprintf("%s/cgi-bin/contactmanage?t=user/index&pageidx=0&type=0&token=%s&lang=zh_CN", MP_WECHAT_URL, m.Token)
	referer := fmt.Sprintf("%s/cgi-bin/home?t=home/index&lang=zh_CN&token=%s", MP_WECHAT_URL, m.Token)

	fmt.Println(url)
	req, req_err := http.NewRequest("GET", url, nil)
	if req_err != nil {
		return wfans, req_err
	}
	c_err := m.AddCookies(req)
	if c_err != nil {
		return wfans, c_err
	}

	req.Header.Add("Referer", referer)
	/*req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/47.0.2526.106 Safari/537.36")
	req.Header.Add("Cache-Control", "no-cache")
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("Pragma", "no-cache")
	req.Header.Add("Upgrade-Insecure-Requests", "1")
	*/
	fmt.Println(referer)
	fmt.Println(url)

	client := &http.Client{}
	resp, client_err := client.Do(req)
	if client_err != nil {
		return wfans, client_err
	}
	defer resp.Body.Close()

	resp_body, read_err := ioutil.ReadAll(resp.Body)
	if read_err != nil {
		return wfans, read_err
	}

	//fmt.Println(string(resp_body))
	matched_strs := REG_GETFANS.FindStringSubmatch(string(resp_body))
	if len(matched_strs) == 0 {
		fmt.Println(string(resp_body))
		return wfans, errors.New("Get friendList error by regexp!")
	}

	//fmt.Println(matched_strs[1])
	json_err := json.Unmarshal([]byte(matched_strs[1]), &wfans)
	if json_err != nil {
		return wfans, errors.New("Parse to json error:" + json_err.Error())
	}

	return wfans, nil
}
