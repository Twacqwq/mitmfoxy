package model

import (
	"bytes"
	"net/http"

	"github.com/google/uuid"
)

type PacketCaptureFlow struct {
	ID       string    `json:"id"`
	Request  *Request  `json:"request"`
	Response *Response `json:"response"`
}

type Request struct {
	Method string      `json:"method"`
	Url    string      `json:"url"`
	Header http.Header `json:"header"`
	Body   []byte      `json:"body"`
}

type Response struct {
	Proto      string         `json:"proto"`
	StatusCode int            `json:"status_code"`
	StatusText string         `json:"status_text"`
	Header     http.Header    `json:"header"`
	Body       []byte         `json:"body"`
	Cookies    []*http.Cookie `json:"cookies"`
}

func BuildPacketCaptureFlow(resp *http.Response, req *http.Request, reqBuf, respBuf *bytes.Buffer) *PacketCaptureFlow {
	return &PacketCaptureFlow{
		ID: uuid.NewString(),
		Request: &Request{
			Method: req.Method,
			Url:    req.URL.String(),
			Header: req.Header,
			Body:   reqBuf.Bytes(),
		},
		Response: &Response{
			Proto:      resp.Proto,
			StatusCode: resp.StatusCode,
			StatusText: http.StatusText(resp.StatusCode),
			Header:     resp.Header,
			Body:       respBuf.Bytes(),
			Cookies:    resp.Cookies(),
		},
	}
}
