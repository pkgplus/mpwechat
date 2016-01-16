package mpwechat

import (
	"bytes"
	//"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	json_sample "github.com/bitly/go-simplejson"
	"io"
	"io/ioutil"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func (m *MpWechat) PrepareMpSendMsg() error {
	if m.PrepareSendFlag {
		return errors.New("have prepared!")
	}
	m.ChanMpSendMsg = make(chan *SendMessage)
	m.ChanMpSendResp = make(chan *WXBaseResp)

	//发送
	go func() {
		for {
			sndmsg := <-m.ChanMpSendMsg
			if err := m.sendMsg(sndmsg); err != nil {
				fmt.Println("ERR:" + err.Error())
			}
		}
	}()
	m.PrepareSendFlag = true
	return nil
}

func (m *MpWechat) GetFirstSendMsg(msg_page *WXMsgPageInfo) *RecvMessage {
	for _, recv_msg := range msg_page.MsgItems.MsgItem {
		if recv_msg.FakeID == m.Uin {
			return &recv_msg
		}
	}

	return &RecvMessage{}
}

func (m *MpWechat) SendText(userid, content string) error {
	sndmsg := &SendMessage{"1", userid, content, "1", false}
	m.ChanMpSendMsg <- sndmsg
	return nil
}

func (m *MpWechat) SendTextBlock(userid, content string, timeout int) error {
	if timeout <= 0 {
		return m.SendText(userid, content)
	}

	//send
	sndmsg := &SendMessage{"1", userid, content, "1", true}
	m.ChanMpSendMsg <- sndmsg

	//wait response
	timeout_chan := make(chan bool, 1)
	go func() {
		time.Sleep(time.Duration(timeout) * time.Second)
		timeout_chan <- true
	}()

	select {
	case send_resp := <-m.ChanMpSendResp:
		if send_resp.Ret == 0 {
			return nil
		} else {
			return errors.New(send_resp.ErrMsg)
		}
	case <-timeout_chan:
		return errors.New("TIMEOUT!")
	}

	return nil
}

func (m *MpWechat) SendImage(userid, filename string) error {
	sndmsg := &SendMessage{"2", userid, filename, "1", false}
	m.ChanMpSendMsg <- sndmsg
	return nil
}

func (m *MpWechat) SendImageBlock(userid, filename string, timeout int) error {
	if timeout <= 0 {
		return m.SendImage(userid, filename)
	}

	sndmsg := &SendMessage{"2", userid, filename, "1", true}
	m.ChanMpSendMsg <- sndmsg

	//wait response
	timeout_chan := make(chan bool, 1)
	go func() {
		time.Sleep(time.Duration(timeout) * time.Second)
		timeout_chan <- true
	}()

	select {
	case send_resp := <-m.ChanMpSendResp:
		if send_resp.Ret == 0 {
			return nil
		} else {
			return errors.New(send_resp.ErrMsg)
		}
	case <-timeout_chan:
		return errors.New("TIMEOUT!")
	}

}

func (m *MpWechat) sendMsg(sndmsg *SendMessage) error {
	referer := MP_WECHAT_URL
	url_req := fmt.Sprintf("%s/cgi-bin/singlesend?t=ajax-response&f=json&token=%s&lang=zh_CN", MP_WECHAT_URL, m.Token)

	v := url.Values{}
	v.Set("token", m.Token)
	v.Set("lang", "zh_CN")
	v.Set("f", "json")
	v.Set("ajax", "1")
	v.Set("random", strconv.FormatFloat(rand.Float64(), 'f', -16, 64))
	v.Set("tofakeid", sndmsg.ToUserID)
	v.Set("imgcode", "")

	if sndmsg.Type == MESSAGE_TYPE_TEXT {
		v.Set("type", MESSAGE_TYPE_TEXT)
		v.Set("Content", sndmsg.Content)
	} else if sndmsg.Type == MESSAGE_TYPE_IMG {
		content_id, upload_err := m.UploadImg(sndmsg.Content)
		if upload_err != nil {
			return upload_err
		}
		v.Set("fileid", content_id)
		v.Set("file_id", content_id)
		v.Set("type", MESSAGE_TYPE_IMG)
	}

	req, req_err := http.NewRequest("POST", url_req, strings.NewReader(v.Encode()))
	if req_err != nil {
		return req_err
	}

	m.AddCookies(req)

	//add header
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Referer", referer)

	//New HTTP CLIENT
	client := &http.Client{}
	resp, client_err := client.Do(req)
	if client_err != nil {
		return client_err
	}
	defer resp.Body.Close()

	resp_body, _ := ioutil.ReadAll(resp.Body)
	//fmt.Println("RECV:" + string(resp_body))

	if sndmsg.Block {
		send_resp := &WXBaseResp{}
		json_err := json.Unmarshal(resp_body, send_resp)
		if json_err != nil {
			return json_err
		}

		m.ChanMpSendResp <- send_resp
	}

	return nil
}

func (m *MpWechat) UploadImg(filename string) (string, error) {
	seq := "1"
	v := url.Values{}
	v.Set("action", "upload_material")
	v.Set("f", "json")
	v.Set("scene", "5")
	v.Set("writetype", "doublewrite")
	v.Set("groupid", "1")
	v.Set("ticket_id", "HiNotice")
	v.Set("token", m.Token)
	v.Set("seq", seq)
	v.Set("svr_time", fmt.Sprintf("%v", time.Now().Unix()))
	v.Set("lang", "zh_CN")

	v.Set("ticket", m.Ticket)

	//url
	post_url := MP_WECHAT_URL + "/cgi-bin/filetransfer?" + v.Encode()

	/*//image content
	  ff, r_err := ioutil.ReadFile(filename)
	  if r_err != nil {
	      return errors.New("read " + filename + " error" + r_err.Error())
	  }
	  content_img := base64.StdEncoding.EncodeToString(ff)
	*/
	img_size := "0"
	ff, r_err := ioutil.ReadFile(filename)
	if r_err != nil {
		return "", errors.New("read " + filename + " error" + r_err.Error())
	}
	img_size = fmt.Sprintf("%v", len(ff))
	//content_img := base64.StdEncoding.EncodeToString(ff)

	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// Add the other fields
	if fw, err := w.CreateFormField("id"); err == nil {
		fw.Write([]byte("WU_FILE_" + seq))
	}
	if fw, err := w.CreateFormField("name"); err == nil {
		fw.Write([]byte(filepath.Base(filename)))
	}
	if fw, err := w.CreateFormField("type"); err == nil {
		fw.Write([]byte("image/jpeg"))
	}
	if fw, err := w.CreateFormField("lastModifiedDate"); err == nil {
		fw.Write([]byte("Fri Jan 01 2016 10:18:03 GMT+0800"))
	}
	if fw, err := w.CreateFormField("size"); err == nil {
		fw.Write([]byte(img_size))
	}

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="file"; filename="%s"`,
			"qrcode_for_gh_30774a893fcd_430.jpg"))
	h.Set("Content-Type", "image/jpeg")
	fw, err := w.CreatePart(h)
	if err != nil {
		return "", err
	}

	f, _ := os.Open(filename)
	if _, err := io.Copy(fw, f); err != nil {
		return "", err
	}
	w.Close()

	//ioutil.WriteFile(`F:\body.txt`, b.Bytes(), 0666)

	//new request
	req, req_err := http.NewRequest("POST", post_url, &b)
	if req_err != nil {
		return "", req_err
	}

	//add header
	req.Header.Set("Connection", "Keep-Alive")
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Origin", MP_WECHAT_URL)
	req.Header.Set("Referer", MP_WECHAT_URL)

	m.AddCookies(req)

	//New HTTP CLIENT
	client := &http.Client{}
	resp, client_err := client.Do(req)
	if client_err != nil {
		return "", client_err
	}
	defer resp.Body.Close()

	//login result
	resp_body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("UPLOAD_IMG:" + string(resp_body))

	//get content
	json_ret, json_err := json_sample.NewJson(resp_body)
	if json_err != nil {
		return "", json_err
	}

	json_map, map_err := json_ret.Map()
	if map_err != nil {
		return "", map_err
	}

	if content_id_v, ok := json_map["content"]; ok {
		content_id, _ := content_id_v.(string)

		return content_id, nil
	}

	err_msg, _ := json_map["err_msg"]
	return "", errors.New(fmt.Sprintf("%v", err_msg))
}
