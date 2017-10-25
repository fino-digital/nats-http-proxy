package natsproxy

import (
	"github.com/labstack/echo"
	"github.com/nats-io/go-nats"
	"net/http/httptest"
	"net/url"
	"net/http"
	"bytes"
	"io"
	"crypto/tls"
	"fmt"
)

type (
	Request struct {
		URL         url.URL            `protobuf:"bytes,1,opt,name=URL,json=uRL" json:"URL,omitempty"`
		Method      string             `protobuf:"bytes,2,opt,name=Method,json=method" json:"Method,omitempty"`
		RemoteAddr  string             `protobuf:"bytes,3,opt,name=RemoteAddr,json=remoteAddr" json:"RemoteAddr,omitempty"`
		Body        []byte             `protobuf:"bytes,4,opt,name=Body,json=body,proto3" json:"Body,omitempty"`
		Form        url.Values         `protobuf:"bytes,5,rep,name=Form,json=form" json:"Form,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
		Header      http.Header        `protobuf:"bytes,6,rep,name=Header,json=header" json:"Header,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
		WebSocketID string             `protobuf:"bytes,7,opt,name=WebSocketID,json=webSocketID" json:"WebSocketID,omitempty"`
	}

	Response struct {
		Status     string // e.g. "200 OK"
		StatusCode int    // e.g. 200
		Proto      string // e.g. "HTTP/1.0"
		ProtoMajor int    // e.g. 1
		ProtoMinor int    // e.g. 0
		Header http.Header
		Body io.ReadCloser
		ContentLength int64
		TransferEncoding []string
		Close bool
		Uncompressed bool
		Trailer http.Header
		Request *Request
		TLS *tls.ConnectionState
	}
)

func CreateNatsProxy(e *echo.Echo, c *nats.EncodedConn) {
	for _, route := range e.Routes() {
		fmt.Println(route)
		for _, method := range []string{
			"GET",
			"POST",
			"PUT",
			"DELETE",
		}{
			c.Subscribe(route.Path, func(subj, reply string, req *Request) error {
				reqMethod := req.Method
				if reqMethod == "" {
					reqMethod = "GET"
				}
				httpReq, err := http.NewRequest(reqMethod, subj, bytes.NewReader(req.Body))
				if err != nil {
					return err
				}
				rec := httptest.NewRecorder()
				ctx := e.NewContext(httpReq, rec)
				e.Router().Find(method, route.Path, ctx)
				e.ServeHTTP(ctx.Response(), ctx.Request())

				return c.Publish(reply, rec.Body)
			})
		}
	}
}
