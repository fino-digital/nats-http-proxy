package natsproxy_test

import (
	"testing"
	"github.com/nats-io/go-nats"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"time"
	"net/http"
	"github.com/fino-digital/nats-http-proxy"
	"github.com/stretchr/testify/assert"
)

func TestEncNatsProxy(t *testing.T) {
	e := echo.New()
	e.HideBanner = true

	e.Use(middleware.Recover())
	e.Use(middleware.Logger())


	nc, _ := nats.Connect(nats.DefaultURL)
	c, _ := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
	//rc := natsproxy.RestNatsEncodedConn{c}

	testData := "woossp"
	e.GET("test/:user/peew", func(c echo.Context) error {
		return c.JSON(http.StatusOK, c.Param("user"))
	})

	/*e.HTTPErrorHandler = func(err error, c echo.Context) {
		t.Log(err, c)
	}*/

	natsproxy.CreateNatsProxy(e, nc)

	var resp string
	err := natsproxy.RestRequestEnc(c, "test/woossp/peew", nil, &resp, time.Second * 5)
	if assert.NoError(t, err) {
		assert.Equal(t, testData, resp)
	}
}

func TestNatsProxy(t *testing.T) {
	e := echo.New()
	e.HideBanner = true

	e.Use(middleware.Recover())
	e.Use(middleware.Logger())


	nc, _ := nats.Connect(nats.DefaultURL)
	//rc := natsproxy.RestNatsEncodedConn{c}

	testData := "heey"
	e.GET("test/:user/peew", func(c echo.Context) error {
		return c.HTML(http.StatusOK, "heey")
	})

	/*e.HTTPErrorHandler = func(err error, c echo.Context) {
		t.Log(err, c)
	}*/

	natsproxy.CreateNatsProxy(e, nc)

	resp, err := natsproxy.RestRequest(nc, "test/woossp/peew", nil, time.Second * 5)
	if assert.NoError(t, err) {
		assert.Equal(t, testData, string(resp.Data[:]))
	}
}