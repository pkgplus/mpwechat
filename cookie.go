package mpwechat

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
)

func (m *MpWechat) AddCookies(req *http.Request) error {
	fi, err := os.Open(m.CookieFile)
	if err != nil {
		return err
	}
	defer fi.Close()

	f_bytes, read_err := ioutil.ReadAll(fi)
	if read_err != nil {
		return read_err
	}

	cookies := make([]*http.Cookie, 0)
	json_err := json.Unmarshal(f_bytes, &cookies)
	if json_err != nil {
		return json_err
	}

	for _, cookie := range cookies {
		//fmt.Println(cookie)
		req.AddCookie(cookie)
	}

	cookie := &http.Cookie{Name: "noticeLoginFlag", Value: "1"}
	req.AddCookie(cookie)
	return nil
}

func (m *MpWechat) ModifyCookies(new_cookie []*http.Cookie) error {
	fi, err := os.Open(m.CookieFile)
	if err != nil {
		return err
	}
	defer fi.Close()

	f_bytes, read_err := ioutil.ReadAll(fi)
	if read_err != nil {
		return read_err
	}

	cookies := make([]*http.Cookie, 0)
	json_err := json.Unmarshal(f_bytes, &cookies)
	if json_err != nil {
		return json_err
	}

	for _, new_cookie := range cookies {
		//fmt.Println(new_cookie.String())
		if new_cookie.Name != "bizuin" {
			continue
		}
		for _, cookie := range cookies {
			if cookie.Name == "bizuin" {
				cookie.Value = new_cookie.Value
				break
			}
		}
	}

	return m.SaveCookies(cookies)
}

func (m *MpWechat) SaveCookies(cookies []*http.Cookie) error {
	cookie_bytes, cookie_err := json.Marshal(cookies)
	if cookie_err != nil {
		return cookie_err
	}
	cookie_w_err := ioutil.WriteFile(m.CookieFile, cookie_bytes, 0666)
	if cookie_w_err != nil {
		return cookie_w_err
	}

	return nil
}
