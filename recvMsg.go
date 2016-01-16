package mpwechat

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

const (
	PREPARE_TYPE_USER_SEND = 1
	PREPARE_TYPE_USER_RECV = 2
	PREPARE_TYPE_USER_ALL  = 3
	REG_STR_USER_MSGS      = `(?s)wx.cgiData = ({"to_nick_name"[^\r\n]+});`
)

var REG_USER_MSGS *regexp.Regexp

func init() {
	REG_USER_MSGS = regexp.MustCompile(REG_STR_USER_MSGS)
}

func (m *MpWechat) PrepareMpRecvMsg(fan *WechatFan, prepare_type int) error {
	//用户发送
	if prepare_type&PREPARE_TYPE_USER_SEND >= 1 {
		if m.ChanUserSendMsg == nil {
			m.ChanUserSendMsg = make(chan *RecvMessage)
		} else {
			return errors.New("you have prepared PREPARE_TYPE_USER_SEND!")
		}
	}

	//用户接收
	if prepare_type&PREPARE_TYPE_USER_RECV >= 1 {
		if m.ChanUserRecvMsg == nil {
			m.ChanUserRecvMsg = make(chan *RecvMessage)
		} else {
			return errors.New("you have prepared PREPARE_TYPE_USER_RECV!")
		}
	}

	//已经监听用户消息
	if fan.WatchedMsgs {
		return nil
	} else {
		//未监听用户消息,接收最新20条消息 并置断点
		m.GetUserRecentMsgs(fan)
	}

	fan.WatchedMsgs = true

	//接收
	go func() {
		for {
			new_msgs, get_err := m.GetUserNewMsgs(fan)
			if get_err != nil {
				fmt.Println(get_err)
				break
			}

			user_send_msgs, user_recv_msgs := new_msgs.SplitUserMsgs(fan.ID)

			//用户发送
			if m.ChanUserSendMsg != nil {
				for _, user_msg := range user_send_msgs {
					m.ChanUserSendMsg <- user_msg
				}
			}
			//用户接收
			if m.ChanUserRecvMsg != nil {
				for _, user_msg := range user_recv_msgs {
					m.ChanUserRecvMsg <- user_msg
				}
			}
			time.Sleep(time.Second)
		}
	}()

	return nil
}

func (m *MpWechat) WaitUserSendMsg(fan *WechatFan, timeout int) (*RecvMessage, error) {
	m.PrepareMpRecvMsg(fan, PREPARE_TYPE_USER_SEND)
	if timeout < 0 {
		user_send_msg := <-m.ChanUserSendMsg
		return user_send_msg, nil
	} else if timeout == 0 {
		if len(m.ChanUserSendMsg) > 0 {
			user_send_msg := <-m.ChanUserSendMsg
			return user_send_msg, nil
		} else {
			return &RecvMessage{}, errors.New("the user didn't send message!")
		}
	} else {
		timeout_chan := make(chan bool, 1)
		go func() {
			time.Sleep(time.Duration(timeout) * time.Second)
			timeout_chan <- true
		}()

		select {
		case user_send_msg := <-m.ChanUserSendMsg:
			return user_send_msg, nil
		case <-timeout_chan:
			return &RecvMessage{}, errors.New("TIMEOUT!")
		}
	}
}

func (m *MpWechat) GetUserRecentMsgs(fan *WechatFan) (*WXMsgPageInfo, error) {
	recent_msgs := &WXMsgPageInfo{}

	url_req := fmt.Sprintf("%s/cgi-bin/singlesendpage", MP_WECHAT_URL)
	v := url.Values{}
	v.Set("t", "message/send")
	v.Set("action", "index")
	v.Set("tofakeid", fan.ID)
	v.Set("token", m.Token)
	v.Set("lang", "zh_CN")
	url_req += "?" + v.Encode()

	req, req_err := http.NewRequest("GET", url_req, nil)
	if req_err != nil {
		return recent_msgs, req_err
	}

	m.AddCookies(req)

	req.Header.Add("Referer", MP_WECHAT_URL)

	//New HTTP CLIENT
	client := &http.Client{}
	resp, client_err := client.Do(req)
	if client_err != nil {
		return recent_msgs, client_err
	}
	defer resp.Body.Close()

	resp_body, _ := ioutil.ReadAll(resp.Body)

	matched_strs := REG_USER_MSGS.FindStringSubmatch(string(resp_body))
	if len(matched_strs) != 2 {
		return recent_msgs, errors.New("Get recent msgs failed by reg:" + REG_STR_USER_MSGS)
	}

	//last messages
	json_err := json.Unmarshal([]byte(matched_strs[1]), recent_msgs)
	if json_err != nil {
		return recent_msgs, json_err
	}

	fan.SetLastMsgInfo(recent_msgs.GetLastMsg())
	fmt.Sprintln("%s(%s) LASTMSGID=%s", fan.NickName, fan.ID, fan.LastMsgInfo.LastMsgID)
	return recent_msgs, nil
}

func (m *MpWechat) GetUserNewMsgs(fan *WechatFan) (*WXMsgPageInfo, error) {
	resp_msg := &WXMsgResp{}

	referer := MP_WECHAT_URL
	url_req := fmt.Sprintf("%s/cgi-bin/singlesendpage", MP_WECHAT_URL)

	v := url.Values{}
	v.Set("tofakeid", fan.ID)
	v.Set("f", "json")
	v.Set("action", "sync")
	v.Set("createtime", fan.LastMsgInfo.CreateTime)
	v.Set("token", m.Token)
	v.Set("lang", "zh_CN")
	v.Set("lastmsgid", fan.LastMsgInfo.LastMsgID)
	v.Set("lastmsgfromfakeid", fan.LastMsgInfo.LastMsgFromFakeID)
	v.Set("ajax", "1")
	v.Set("random", strconv.FormatFloat(rand.Float64(), 'f', -16, 64))
	url_req += "?" + v.Encode()

	req, req_err := http.NewRequest("GET", url_req, nil)
	if req_err != nil {
		return &resp_msg.PageInfo, req_err
	}

	m.AddCookies(req)
	req.Header.Add("Referer", referer)

	//New HTTP CLIENT
	client := &http.Client{}
	resp, client_err := client.Do(req)
	if client_err != nil {
		return &resp_msg.PageInfo, client_err
	}
	defer resp.Body.Close()

	resp_body, _ := ioutil.ReadAll(resp.Body)
	json_err := json.Unmarshal(resp_body, resp_msg)
	if json_err != nil {
		return &resp_msg.PageInfo, json_err
	}
	//fmt.Println(resp_msg)

	//更新断点
	if resp_msg.PageInfo.GetLastMsg().ID > 0 {
		fan.SetLastMsgInfo(resp_msg.PageInfo.GetLastMsg())
	}

	return &resp_msg.PageInfo, nil
}
