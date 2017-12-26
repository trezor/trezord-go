//-----------------------------------------------------------------------------
/*

Test functions for libusb-1.0

*/
//-----------------------------------------------------------------------------

package libusb

import (
	"log"
	"os"
	"testing"
	"unsafe"
)

//-----------------------------------------------------------------------------

var logger = log.New(os.Stdout, "", log.Lshortfile)

//-----------------------------------------------------------------------------

func Test_Error_Name(t *testing.T) {
	if Error_Name(ERROR_BUSY) != "LIBUSB_ERROR_BUSY" {
		t.Error("FAIL")
	}
}

func Test_Device_List(t *testing.T) {
	var ctx Context
	err := Init(&ctx)
	defer Exit(ctx)
	if err != nil {
		t.Error("FAIL")
	}

	list, err := Get_Device_List(ctx)
	if err != nil {
		t.Error("FAIL")
	}

	for _, dev := range list {
		dd, err := Get_Device_Descriptor(dev)
		if err != nil {
			t.Error("FAIL")
		}
		path := make([]byte, 8)
		path, err = Get_Port_Numbers(dev, path)
		if err != nil {
			t.Error("FAIL")
		}
		logger.Printf("Bus %03d Device %03d: ID %04x:%04x", Get_Bus_Number(dev), Get_Device_Address(dev), dd.IdVendor, dd.IdProduct)
		logger.Printf("device %08x parent %08x", unsafe.Pointer(dev), unsafe.Pointer(Get_Parent(dev)))
		logger.Printf("%v %d", path, Get_Port_Number(dev))
	}

	Free_Device_List(list, 1)
}

func Test_Init_Exit(t *testing.T) {
	var ctx Context
	err := Init(&ctx)
	defer Exit(ctx)
	if err != nil {
		t.Error("FAIL")
	}
}

//-----------------------------------------------------------------------------
