package ctrl

import (
	"sync"

	"github.com/adlindo/gocom"
	"github.com/kodernubie/wabot/svc"
)

type DeviceCtrl struct {
}

var deviceCtrl *DeviceCtrl
var deviceCtrlOnce sync.Once

func GetDeviceCtrl() *DeviceCtrl {

	deviceCtrlOnce.Do(func() {

		deviceCtrl = &DeviceCtrl{}
	})

	return deviceCtrl
}

func (o *DeviceCtrl) Init() {

	gocom.POST("/device/new", o.newDevice)
}

func (o *DeviceCtrl) newDevice(ctx gocom.Context) error {

	ret, err := svc.GetWASvc().NeWDevice()

	if err != nil {
		return ctx.SendError(gocom.NewError(1001, "Create device error : "+err.Error()))
	}

	return ctx.SendResult(ret)
}
