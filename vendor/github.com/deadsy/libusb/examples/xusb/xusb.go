package main

import (
	"encoding/hex"
	"fmt"
	"os"

	"github.com/deadsy/libusb"
)

var extra_info bool = true

const (
	USE_GENERIC = iota
	USE_PS3
	USE_XBOX
	USE_SCSI
	USE_HID
)

var test_mode int

//-----------------------------------------------------------------------------

// HID Class-Specific Requests values. See section 7.2 of the HID specifications
const HID_GET_REPORT = 0x01
const HID_GET_IDLE = 0x02
const HID_GET_PROTOCOL = 0x03
const HID_SET_REPORT = 0x09
const HID_SET_IDLE = 0x0A
const HID_SET_PROTOCOL = 0x0B
const HID_REPORT_TYPE_INPUT = 0x01
const HID_REPORT_TYPE_OUTPUT = 0x02
const HID_REPORT_TYPE_FEATURE = 0x03

/*

func get_hid_record_size(hid_report_descriptor []byte, record_type int) int {
{
	//uint8_t i, j = 0;
	//uint8_t offset;
	//int record_size[3] = {0, 0, 0};
	//int nb_bits = 0, nb_items = 0;
	//bool found_record_marker;

	found_record_marker := false
	for i := hid_report_descriptor[0]+1; i < len(hid_report_descriptor); i += offset {
		offset = (hid_report_descriptor[i]&0x03) + 1
		if offset == 4 {
			offset = 5
    }
		switch hid_report_descriptor[i] & 0xFC {
		case 0x74:	// bitsize
			nb_bits = hid_report_descriptor[i+1];

		case 0x94:	// count
			nb_items = 0
			for j:=1; j<offset; j++ {
				nb_items = ((uint32_t)hid_report_descriptor[i+j]) << (8*(j-1))
			}

		case 0x80:	// input
			found_record_marker = true
			j = 0

		case 0x90:	// output
			found_record_marker = true
			j = 1

		case 0xb0:	// feature
			found_record_marker = true
			j = 2

		case 0xC0:	// end of collection
			nb_items = 0
			nb_bits = 0

		default:
			continue
		}
		if found_record_marker {
			found_record_marker = false
			record_size[j] += nb_items*nb_bits
		}
	}

	if (record_type < HID_REPORT_TYPE_INPUT) || (record_type > HID_REPORT_TYPE_FEATURE) {
		return 0
	}
	return record_size[record_type - HID_REPORT_TYPE_INPUT]+7)/8
}

*/

func test_hid(handle libusb.Device_Handle, endpoint_in uint8) int {
	//int r, size, descriptor_size;
	hid_report_descriptor := make([]byte, 256)
	//uint8_t *report_buffer;
	//FILE *fd;

	fmt.Printf("\nReading HID Report Descriptors:\n")
	hid_report_descriptor, err := libusb.Control_Transfer(handle, libusb.ENDPOINT_IN|libusb.REQUEST_TYPE_STANDARD|libusb.RECIPIENT_INTERFACE,
		libusb.REQUEST_GET_DESCRIPTOR, libusb.DT_REPORT<<8, 0, hid_report_descriptor, 1000)
	if err != nil {
		fmt.Printf("   Failed\n")
		return -1
	}
	fmt.Printf("%s\n", hex.Dump(hid_report_descriptor))

	/*

		size = get_hid_record_size(hid_report_descriptor, descriptor_size, HID_REPORT_TYPE_FEATURE);
		if (size <= 0) {
			printf("\nSkipping Feature Report readout (None detected)\n");
		} else {
			report_buffer = (uint8_t*) calloc(size, 1);
			if (report_buffer == NULL) {
				return -1;
			}

			printf("\nReading Feature Report (length %d)...\n", size);
			r = libusb_control_transfer(handle, LIBUSB_ENDPOINT_IN|LIBUSB_REQUEST_TYPE_CLASS|LIBUSB_RECIPIENT_INTERFACE,
				HID_GET_REPORT, (HID_REPORT_TYPE_FEATURE<<8)|0, 0, report_buffer, (uint16_t)size, 5000);
			if (r >= 0) {
				display_buffer_hex(report_buffer, size);
			} else {
				switch(r) {
				case LIBUSB_ERROR_NOT_FOUND:
					printf("   No Feature Report available for this device\n");
					break;
				case LIBUSB_ERROR_PIPE:
					printf("   Detected stall - resetting pipe...\n");
					libusb_clear_halt(handle, 0);
					break;
				default:
					printf("   Error: %s\n", libusb_strerror((enum libusb_error)r));
					break;
				}
			}
			free(report_buffer);
		}

		size = get_hid_record_size(hid_report_descriptor, descriptor_size, HID_REPORT_TYPE_INPUT);
		if (size <= 0) {
			printf("\nSkipping Input Report readout (None detected)\n");
		} else {
			report_buffer = (uint8_t*) calloc(size, 1);
			if (report_buffer == NULL) {
				return -1;
			}

			printf("\nReading Input Report (length %d)...\n", size);
			r = libusb_control_transfer(handle, LIBUSB_ENDPOINT_IN|LIBUSB_REQUEST_TYPE_CLASS|LIBUSB_RECIPIENT_INTERFACE,
				HID_GET_REPORT, (HID_REPORT_TYPE_INPUT<<8)|0x00, 0, report_buffer, (uint16_t)size, 5000);
			if (r >= 0) {
				display_buffer_hex(report_buffer, size);
			} else {
				switch(r) {
				case LIBUSB_ERROR_TIMEOUT:
					printf("   Timeout! Please make sure you act on the device within the 5 seconds allocated...\n");
					break;
				case LIBUSB_ERROR_PIPE:
					printf("   Detected stall - resetting pipe...\n");
					libusb_clear_halt(handle, 0);
					break;
				default:
					printf("   Error: %s\n", libusb_strerror((enum libusb_error)r));
					break;
				}
			}

			// Attempt a bulk read from endpoint 0 (this should just return a raw input report)
			printf("\nTesting interrupt read using endpoint %02X...\n", endpoint_in);
			r = libusb_interrupt_transfer(handle, endpoint_in, report_buffer, size, &size, 5000);
			if (r >= 0) {
				display_buffer_hex(report_buffer, size);
			} else {
				printf("   %s\n", libusb_strerror((enum libusb_error)r));
			}

			free(report_buffer);
		}

	*/

	return 0
}

//-----------------------------------------------------------------------------

func print_device_cap(dev_cap *libusb.BOS_Dev_Capability_Descriptor) {
	/*
		switch(dev_cap->bDevCapabilityType) {
		case LIBUSB_BT_USB_2_0_EXTENSION: {
			struct libusb_usb_2_0_extension_descriptor *usb_2_0_ext = NULL;
			libusb_get_usb_2_0_extension_descriptor(NULL, dev_cap, &usb_2_0_ext);
			if (usb_2_0_ext) {
				fmt.Printf("    USB 2.0 extension:\n");
				fmt.Pprintf("      attributes             : %02X\n", usb_2_0_ext->bmAttributes);
				libusb_free_usb_2_0_extension_descriptor(usb_2_0_ext);
			}
			break;
		}
		case LIBUSB_BT_SS_USB_DEVICE_CAPABILITY: {
			struct libusb_ss_usb_device_capability_descriptor *ss_usb_device_cap = NULL;
			libusb_get_ss_usb_device_capability_descriptor(NULL, dev_cap, &ss_usb_device_cap);
			if (ss_usb_device_cap) {
				fmt.Pprintf("    USB 3.0 capabilities:\n");
				fmt.Pprintf("      attributes             : %02X\n", ss_usb_device_cap->bmAttributes)
				fmt.Pprintf("      supported speeds       : %04X\n", ss_usb_device_cap->wSpeedSupported)
				fmt.Pprintf("      supported functionality: %02X\n", ss_usb_device_cap->bFunctionalitySupport)
				libusb_free_ss_usb_device_capability_descriptor(ss_usb_device_cap)
			}
			break;
		}
		case LIBUSB_BT_CONTAINER_ID: {
			struct libusb_container_id_descriptor *container_id = NULL;
			libusb_get_container_id_descriptor(NULL, dev_cap, &container_id);
			if (container_id) {
				fmt.Pprintf("    Container ID:\n      %s\n", uuid_to_string(container_id->ContainerID))
				libusb_free_container_id_descriptor(container_id);
			}
			break;
		}
		default:
			fmt.Pprintf("    Unknown BOS device capability %02x:\n", dev_cap->bDevCapabilityType)
		}
	*/
}

func test_device(vid uint16, pid uint16) int {
	port_path := make([]byte, 8)
	speed_name := [5]string{
		"Unknown",
		"1.5 Mbit/s (USB LowSpeed)",
		"12 Mbit/s (USB FullSpeed)",
		"480 Mbit/s (USB HighSpeed)",
		"5000 Mbit/s (USB SuperSpeed)",
	}

	string_index := make([]byte, 3) // indexes of the string descriptors
	// default IN and OUT endpoints
	var endpoint_in uint8
	var endpoint_out uint8

	fmt.Printf("Opening device %04X:%04X...\n", vid, pid)
	handle := libusb.Open_Device_With_VID_PID(nil, vid, pid)
	if handle == nil {
		fmt.Fprintf(os.Stderr, "  Failed.\n")
		return -1
	}

	dev := libusb.Get_Device(handle)
	bus := libusb.Get_Bus_Number(dev)

	if extra_info {
		port_path, err := libusb.Get_Port_Numbers(dev, port_path)
		if err == nil {
			fmt.Printf("\nDevice properties:\n")
			fmt.Printf("        bus number: %d\n", bus)
			fmt.Printf("         port path: %d", port_path[0])
			for i := 1; i < len(port_path); i++ {
				fmt.Printf("->%d", port_path[i])
			}
			fmt.Printf(" (from root hub)\n")
		}
		r := libusb.Get_Device_Speed(dev)
		if (r < 0) || (r > 4) {
			r = 0
		}
		fmt.Printf("             speed: %s\n", speed_name[r])
	}

	fmt.Printf("\nReading device descriptor:\n")
	dev_desc, err := libusb.Get_Device_Descriptor(dev)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return -1
	}

	fmt.Printf("            length: %d\n", dev_desc.BLength)
	fmt.Printf("      device class: %d\n", dev_desc.BDeviceClass)
	fmt.Printf("               S/N: %d\n", dev_desc.ISerialNumber)
	fmt.Printf("           VID:PID: %04X:%04X\n", dev_desc.IdVendor, dev_desc.IdProduct)
	fmt.Printf("         bcdDevice: %04X\n", dev_desc.BcdDevice)
	fmt.Printf("   iMan:iProd:iSer: %d:%d:%d\n", dev_desc.IManufacturer, dev_desc.IProduct, dev_desc.ISerialNumber)
	fmt.Printf("          nb confs: %d\n", dev_desc.BNumConfigurations)
	// Copy the string descriptors for easier parsing
	string_index[0] = dev_desc.IManufacturer
	string_index[1] = dev_desc.IProduct
	string_index[2] = dev_desc.ISerialNumber

	fmt.Printf("\nReading BOS descriptor: ")
	bos_desc, err := libusb.Get_BOS_Descriptor(handle)
	if err == nil {
		fmt.Printf("%d caps\n", len(bos_desc.Dev_capability))
		for i := 0; i < len(bos_desc.Dev_capability); i++ {
			print_device_cap(bos_desc.Dev_capability[i])
		}
		libusb.Free_BOS_Descriptor(bos_desc)
	} else {
		fmt.Printf("no descriptor\n")
	}

	fmt.Printf("\nReading first configuration descriptor:\n")
	conf_desc, err := libusb.Get_Config_Descriptor(dev, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return -1
	}

	nb_ifaces := len(conf_desc.Interface)
	fmt.Printf("             nb interfaces: %d\n", nb_ifaces)
	if nb_ifaces > 0 {
		//first_iface := conf_desc.Interface[0].Altsetting[0].BInterfaceNumber
	}
	for i := 0; i < nb_ifaces; i++ {
		fmt.Printf("              interface[%d]: id = %d\n", i, conf_desc.Interface[i].Altsetting[0].BInterfaceNumber)
		for j := 0; j < conf_desc.Interface[i].Num_altsetting; j++ {
			fmt.Printf("interface[%d].altsetting[%d]: num endpoints = %d\n", i, j, conf_desc.Interface[i].Altsetting[j].BNumEndpoints)
			fmt.Printf("   Class.SubClass.Protocol: %02X.%02X.%02X\n",
				conf_desc.Interface[i].Altsetting[j].BInterfaceClass,
				conf_desc.Interface[i].Altsetting[j].BInterfaceSubClass,
				conf_desc.Interface[i].Altsetting[j].BInterfaceProtocol)
			if (conf_desc.Interface[i].Altsetting[j].BInterfaceClass == libusb.CLASS_MASS_STORAGE) &&
				((conf_desc.Interface[i].Altsetting[j].BInterfaceSubClass == 0x01) ||
					(conf_desc.Interface[i].Altsetting[j].BInterfaceSubClass == 0x06)) &&
				(conf_desc.Interface[i].Altsetting[j].BInterfaceProtocol == 0x50) {
				// Mass storage devices that can use basic SCSI commands
				test_mode = USE_SCSI
			}

			for k := 0; k < int(conf_desc.Interface[i].Altsetting[j].BNumEndpoints); k++ {
				endpoint := conf_desc.Interface[i].Altsetting[j].Endpoint[k]
				fmt.Printf("       endpoint[%d].address: %02X\n", k, endpoint.BEndpointAddress)
				// Use the first interrupt or bulk IN/OUT endpoints as default for testing
				if (endpoint.BmAttributes&libusb.TRANSFER_TYPE_MASK)&(libusb.TRANSFER_TYPE_BULK|libusb.TRANSFER_TYPE_INTERRUPT) != 0 {
					if endpoint.BEndpointAddress&libusb.ENDPOINT_IN != 0 {
						if endpoint_in == 0 {
							endpoint_in = endpoint.BEndpointAddress
						}
					} else {
						if endpoint_out == 0 {
							endpoint_out = endpoint.BEndpointAddress
						}
					}
				}
				fmt.Printf("           max packet size: %04X\n", endpoint.WMaxPacketSize)
				fmt.Printf("          polling interval: %02X\n", endpoint.BInterval)
				ep_comp, _ := libusb.Get_SS_Endpoint_Companion_Descriptor(nil, endpoint)
				if ep_comp != nil {
					fmt.Printf("                 max burst: %02X   (USB 3.0)\n", ep_comp.BMaxBurst)
					fmt.Printf("        bytes per interval: %04X (USB 3.0)\n", ep_comp.WBytesPerInterval)
					libusb.Free_SS_Endpoint_Companion_Descriptor(ep_comp)
				}
			}
		}
	}
	libusb.Free_Config_Descriptor(conf_desc)

	libusb.Set_Auto_Detach_Kernel_Driver(handle, true)
	for iface := 0; iface < nb_ifaces; iface++ {
		fmt.Printf("\nClaiming interface %d...\n", iface)

		err := libusb.Claim_Interface(handle, iface)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			return -1
		}
	}

	str := make([]byte, 128)

	fmt.Printf("\nReading string descriptors:\n")
	for i := 0; i < 3; i++ {
		if string_index[i] == 0 {
			continue
		}
		str, err := libusb.Get_String_Descriptor_ASCII(handle, string_index[i], str)
		if err == nil {
			fmt.Printf("   String (0x%02X): \"%s\"\n", string_index[i], string(str))
		}
	}

	// Read the OS String Descriptor
	str, err = libusb.Get_String_Descriptor_ASCII(handle, 0xEE, str)
	if err == nil {
		fmt.Printf("   String (0x%02X): \"%s\"\n", 0xEE, string(str))
		// If this is a Microsoft OS String Descriptor,
		// attempt to read the WinUSB extended Feature Descriptors
		//if (strncmp(string, "MSFT100", 7) == 0)
		//	read_ms_winsub_feature_descriptors(handle, string[7], first_iface);
	}

	test_hid(handle, endpoint_in)

	/*
		switch(test_mode) {
		case USE_PS3:
			CALL_CHECK(display_ps3_status(handle));
			break;
		case USE_XBOX:
			CALL_CHECK(display_xbox_status(handle));
			CALL_CHECK(set_xbox_actuators(handle, 128, 222));
			msleep(2000);
			CALL_CHECK(set_xbox_actuators(handle, 0, 0));
			break;
		case USE_HID:
			test_hid(handle, endpoint_in);
			break;
		case USE_SCSI:
			CALL_CHECK(test_mass_storage(handle, endpoint_in, endpoint_out));
		case USE_GENERIC:
			break;
		}
	*/

	fmt.Printf("\n")
	for iface := 0; iface < nb_ifaces; iface++ {
		fmt.Printf("Releasing interface %d...\n", iface)
		err := libusb.Release_Interface(handle, iface)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			return -1
		}
	}

	fmt.Printf("Closing device...\n")
	libusb.Close(handle)
	return 0
}

//-----------------------------------------------------------------------------

func main() {

	version := libusb.Get_Version()
	fmt.Printf("Using libusb v%d.%d.%d.%d\n\n", version.Major, version.Minor, version.Micro, version.Nano)
	err := libusb.Init(nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(-1)
	}

	test_device(0x045e, 0x00a4)
	//test_device(0x944, 0x115)

	libusb.Exit(nil)
	os.Exit(0)
}

//-----------------------------------------------------------------------------
