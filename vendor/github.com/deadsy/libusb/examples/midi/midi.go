//-----------------------------------------------------------------------------
/*

Simple driver for a MIDI keyboard

Open the device with the provided VID/PID.
Bulk read from the first input endpoint.
Interpret the data as USB MIDI events.

*/
//-----------------------------------------------------------------------------

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/deadsy/libusb"
)

type ep_info struct {
	itf int
	ep  *libusb.Endpoint_Descriptor
}

var quit bool = false

const NOTES_IN_OCTAVE = 12

func midi_note_name(note byte, mode string) string {
	sharps := [NOTES_IN_OCTAVE]string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}
	flats := [NOTES_IN_OCTAVE]string{"C", "Db", "D", "Eb", "E", "F", "Gb", "G", "Ab", "A", "Bb", "B"}
	note %= NOTES_IN_OCTAVE
	if mode == "#" {
		return sharps[note]
	}
	return flats[note]
}

// return a note name with sharp and flat forms
func midi_full_note_name(note byte) string {
	s_name := midi_note_name(note, "#")
	f_name := midi_note_name(note, "b")
	if s_name != f_name {
		return fmt.Sprintf("%s/%s", s_name, f_name)
	}
	return s_name
}

func midi_event(event []byte) {
	if event[0] == 0 &&
		event[1] == 0 &&
		event[2] == 0 &&
		event[3] == 0 {
		//ignore
		return
	}
	ch := (event[0] >> 4) & 15
	switch event[0] & 15 {
	case 8:
		fmt.Printf("ch %d note off %02x %02x %02x %s\n", ch, event[1], event[2], event[3], midi_full_note_name(event[2]))
	case 9:
		fmt.Printf("ch %d note on  %02x %02x %02x %s\n", ch, event[1], event[2], event[3], midi_full_note_name(event[2]))
	case 11:
		fmt.Printf("ch %d ctrl     %02x %02x %02x\n", ch, event[1], event[2], event[3])
	case 14:
		fmt.Printf("ch %d pitch    %02x %02x %02x\n", ch, event[1], event[2], event[3])
	default:
		fmt.Printf("ch %d ?        %02x %02x %02x\n", ch, event[1], event[2], event[3])
	}
}

func midi_device(ctx libusb.Context, vid uint16, pid uint16) {
	fmt.Printf("Opening device %04X:%04X ", vid, pid)
	hdl := libusb.Open_Device_With_VID_PID(ctx, vid, pid)
	if hdl == nil {
		fmt.Printf("failed (do you have permission?)\n")
		return
	}
	fmt.Printf("ok\n")
	defer libusb.Close(hdl)

	dev := libusb.Get_Device(hdl)
	dd, err := libusb.Get_Device_Descriptor(dev)
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}

	// record information on input endpoints
	ep_in := make([]ep_info, 0, 1)
	midi_found := false

	for i := 0; i < int(dd.BNumConfigurations); i++ {
		cd, err := libusb.Get_Config_Descriptor(dev, uint8(i))
		if err != nil {
			fmt.Printf("%s\n", err)
			return
		}
		// iterate across endpoints
		for _, itf := range cd.Interface {
			for _, id := range itf.Altsetting {
				if id.BInterfaceClass == libusb.CLASS_AUDIO && id.BInterfaceSubClass == 3 {
					midi_found = true
				}
				for _, ep := range id.Endpoint {
					if ep.BEndpointAddress&libusb.ENDPOINT_IN != 0 {
						ep_in = append(ep_in, ep_info{itf: i, ep: ep})
					}
				}
			}
		}

		libusb.Free_Config_Descriptor(cd)
	}

	if midi_found == false || len(ep_in) == 0 {
		fmt.Printf("no midi inputs found\n")
		return
	}

	fmt.Printf("num input endpoints %d\n", len(ep_in))
	libusb.Set_Auto_Detach_Kernel_Driver(hdl, true)

	// claim the interface
	err = libusb.Claim_Interface(hdl, ep_in[0].itf)
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}
	defer libusb.Release_Interface(hdl, ep_in[0].itf)

	data := make([]byte, ep_in[0].ep.WMaxPacketSize)
	for quit == false {
		data, err := libusb.Bulk_Transfer(hdl, ep_in[0].ep.BEndpointAddress, data, 1000)
		if err == nil {
			for i := 0; i < len(data); i += 4 {
				// each midi event is 4 bytes
				midi_event(data[i : i+4])
			}
		}
	}
}

func midi_main() int {
	var ctx libusb.Context
	err := libusb.Init(&ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return -1
	}
	defer libusb.Exit(ctx)

	midi_device(ctx, 0x0944, 0x0115) // Korg Nano Key 2
	//midi_device(ctx, 0x041e, 0x3f0e) // Creative Technology, E-MU XMidi1X1 Tab

	return 0
}

func main() {

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		fmt.Printf("\n%s\n", sig)
		quit = true
	}()

	os.Exit(midi_main())
}
