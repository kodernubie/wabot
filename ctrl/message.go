package ctrl

import (
	"sync"

	"github.com/adlindo/gocom"
)

type MessageCtrl struct {
}

var messageCtrl *MessageCtrl
var messageCtrlOnce sync.Once

func GetMessageCtrl() *MessageCtrl {

	messageCtrlOnce.Do(func() {

		messageCtrl = &MessageCtrl{}
	})

	return messageCtrl
}

func (o *MessageCtrl) Init() {

	gocom.GET("/test", o.test)
}

func (o *MessageCtrl) test(ctx gocom.Context) error {

	return ctx.SendResult(true)
}
