package model

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	_ = iota
	NotificationRequestTypeJSON
	NotificationRequestTypeForm
)

const (
	_ = iota
	NotificationRequestMethodGET
	NotificationRequestMethodPOST
)

type Notification struct {
	Common
	Name          string
	URL           string
	RequestMethod int
	RequestType   int
	RequestBody   string `gorm:"type:longtext" `
	VerifySSL     *bool
}

func (n *Notification) reqURL(message string) string {
	return replaceParamsInString(n.URL, message)
}

func (n *Notification) reqBody(message string) (string, error) {
	if n.RequestMethod == NotificationRequestMethodGET {
		return "", nil
	}
	switch n.RequestType {
	case NotificationRequestTypeJSON:
		return replaceParamsInString(n.RequestBody, message), nil
	case NotificationRequestTypeForm:
		var data map[string]string
		if err := json.Unmarshal([]byte(n.RequestBody), &data); err != nil {
			return "", err
		}
		params := url.Values{}
		for k, v := range data {
			params.Add(k, replaceParamsInString(v, message))
		}
		return params.Encode(), nil
	}
	return "", errors.New("不支持的请求类型")
}

func (n *Notification) reqContentType() string {
	if n.RequestMethod == NotificationRequestMethodGET {
		return ""
	}
	if n.RequestType == NotificationRequestTypeForm {
		return "application/x-www-form-urlencoded"
	} else {
		return "application/json"
	}
}

func (n *Notification) Send(message string) error {
	var verifySSL bool

	if n.VerifySSL != nil && *n.VerifySSL {
		verifySSL = true
	}

	transCfg := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: verifySSL},
	}
	client := &http.Client{Transport: transCfg, Timeout: time.Minute * 10}

	reqBody, err := n.reqBody(message)

	var resp *http.Response

	if err == nil {
		if n.RequestMethod == NotificationRequestMethodGET {
			resp, err = client.Get(n.reqURL(message))
		} else {
			resp, err = client.Post(n.reqURL(message), n.reqContentType(), strings.NewReader(reqBody))
		}
	}

	if err == nil && (resp.StatusCode < 200 || resp.StatusCode > 299) {
		err = fmt.Errorf("%d %s", resp.StatusCode, resp.Status)
	}

	return err
}

func replaceParamsInString(str string, message string) string {
	str = strings.ReplaceAll(str, "#NEZHA#", message)
	return str
}
