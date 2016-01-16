package mpwechat

import (
//"encoding/json"
)

const (
	MESSAGE_TYPE_TEXT = "1"
	MESSAGE_TYPE_IMG  = "2"
)

type SendMessage struct {
	Type     string
	ToUserID string
	Content  string
	MsgID    string
	Block    bool
}

type WXMsgResp struct {
	BaseResp WXBaseResp    `json:"base_resp"`
	PageInfo WXMsgPageInfo `json:"page_info,omitempty"`
}

type WXBaseResp struct {
	Ret    int    `json:"ret"`
	ErrMsg string `json:"err_msg"`
}

type WXMsgPageInfo struct {
	ToNickName string `json:"to_nick_name"`
	MsgItems   struct {
		MsgItem []RecvMessage `json:"msg_item"`
	} `json:"msg_items"`
}

type RecvMessage struct {
	ID           int    `json:"id"`
	Type_num     int    `json:"type"`
	FakeID       string `json:"fakeid"`
	NickName     string `json:"nick_name"`
	DateTime     int    `json:"date_time"`
	Content      string `json:"content,omitempty"`
	Source       string `json:"source"`
	MsgStatus    int    `json:"msg_status"`
	HasReply     int    `json:"has_reply"`
	RefuseReason string `json:"refuse_reason"`
	ToUin        string `json:"to_uin"`
	WxHeadingUrl string `json:"wx_headimg_url"`
}

func (w *WXMsgPageInfo) GetLastMsg() *RecvMessage {
	if len(w.MsgItems.MsgItem) >= 1 {
		return &w.MsgItems.MsgItem[0]
	} else {
		return &RecvMessage{}
	}
}

func (w *WXMsgPageInfo) SplitUserMsgs(userid string) ([]*RecvMessage, []*RecvMessage) {
	user_sended := make([]*RecvMessage, 0)
	user_recved := make([]*RecvMessage, 0)

	for _, recv_msg := range w.MsgItems.MsgItem {
		//json_b, _ := json.Marshal(recv_msg)
		//目标不为用户本身的消息为用户发送消息
		if recv_msg.ToUin != userid {
			user_sended = append(user_sended, &recv_msg)
		} else {
			user_recved = append(user_recved, &recv_msg)
		}
	}
	return user_sended, user_recved
}
