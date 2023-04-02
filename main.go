package main

import (
	"fmt"

	"github.com/adlindo/gocom"
	"github.com/kodernubie/wabot/ctrl"
	"github.com/kodernubie/wabot/svc"
)

func main() {

	fmt.Println("====>> WA BOT <<====")

	gocom.AddCtrl(ctrl.GetDeviceCtrl())
	gocom.AddCtrl(ctrl.GetMessageCtrl())

	svc.GetWASvc()

	gocom.Start()
}
