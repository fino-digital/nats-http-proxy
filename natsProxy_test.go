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
	"log"
	natsproxy2 "github.com/sohlich/nats-proxy"
	"encoding/json"
)

func TestNatsProxy(t *testing.T) {
	e := echo.New()
	e.HideBanner = true

	e.Use(middleware.Recover())
	e.Use(middleware.Logger())


	// Set up our natsconnections
	c, _ := nats.Connect(nats.DefaultURL)
	nc, _ := nats.NewEncodedConn(c, nats.JSON_ENCODER)
	rc := natsproxy.RestNatsEncConn{nc}

	testParam := "testParam"
	testQuery := "testQuery"
	testHeader := "testHeaderKey"
	testHeaderValue := "testHeaderValue"

	e.POST("test/:testParam/peew", func(c echo.Context) error {
		c.Request().ParseForm()
		return c.JSON(http.StatusOK, []string{
			c.Param("testParam"),
			c.QueryParam("peew"),
			c.Request().Header.Get(testHeader),
		})
	})

	// Automatically proxy all routes
	natsproxy.CreateNatsProxy(e, c)

	// nats/ prefix is important!
	req, err := http.NewRequest("POST","https://peew.com/nats/test/"+testParam+"/peew?peew="+testQuery, nil)

	req.Header.Set(testHeader, testHeaderValue)
	// Send a fake http over nats and auto unmarshal the result

	pewReq := natsproxy2.NewRequest()
	pewReq.FromHTTP(req)
	jsonReq, err := json.Marshal(pewReq)
	log.Println(string(jsonReq[:]))
	var res []string
	err = rc.RestRequest(req.URL.Path, req , &res, time.Second * 45)
	if assert.NoError(t, err) {
		assert.Equal(t, []string{testParam, testQuery, testHeaderValue}, res)

	}
}