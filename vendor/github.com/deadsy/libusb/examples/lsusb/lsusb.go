package main

import (
	"fmt"
	"log"

	"github.com/deadsy/libusb"
)

func main() {

	var ctx libusb.Context
	err := libusb.Init(&ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer libusb.Exit(ctx)

	list, err := libusb.Get_Device_List(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer libusb.Free_Device_List(list, 1)

	for _, dev := range list {
		dd, err := libusb.Get_Device_Descriptor(dev)
		if err != nil {
			log.Fatal(err)
		}
		path := make([]byte, 8)
		path, err = libusb.Get_Port_Numbers(dev, path)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Bus %03d Device %03d: ID %04x:%04x", libusb.Get_Bus_Number(dev), libusb.Get_Device_Address(dev), dd.IdVendor, dd.IdProduct)
		fmt.Printf("%v %d\n", path, libusb.Get_Port_Number(dev))
	}
}
