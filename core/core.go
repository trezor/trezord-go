package core

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/trezor/trezord-go/memorywriter"
	"github.com/trezor/trezord-go/wire"
)

// Package with "core logic" of device listing
// and dealing with sessions, mutexes, ...
//
// USB package is not imported for efficiency
// reasons - USB package uses imports /usb/lowlevel and
// /usb/lowlevel uses cgo, so it takes about 25 seconds to build;
// so building just this package  on its own
// takes less seconds when we dont import USB
// package and use abstract interfaces instead

// USB* interfaces are implemented in usb package

type USBBus interface {
	Enumerate() ([]USBInfo, error)
	Connect(
		path string,
		debug bool, // debug link
		reset bool, // reset is optional, to prevent reseting calls
	) (USBDevice, error)
	Has(path string) bool
}

type DeviceType int

const (
	TypeT1Hid        DeviceType = 0
	TypeT1Webusb     DeviceType = 1
	TypeT1WebusbBoot DeviceType = 2
	TypeT2           DeviceType = 3
	TypeT2Boot       DeviceType = 4
	TypeEmulator     DeviceType = 5
)

type USBInfo struct {
	Path      string
	VendorID  int
	ProductID int
	Type      DeviceType
	Debug     bool // has debug enabled?
}

type USBDevice interface {
	io.ReadWriter
	Close(disconnected bool) error
}

type session struct {
	path string
	id   string
	dev  USBDevice
	call int32 // atomic
}

type EnumerateEntry struct {
	Path    string     `json:"path"`
	Vendor  int        `json:"vendor"`
	Product int        `json:"product"`
	Type    DeviceType `json:"-"`     // used only in status page, not in JSON
	Debug   bool       `json:"debug"` // has debug enabled?

	Session      *string `json:"session"`
	DebugSession *string `json:"debugSession"`
}

type EnumerateEntries []EnumerateEntry

func (entries EnumerateEntries) Len() int {
	return len(entries)
}
func (entries EnumerateEntries) Less(i, j int) bool {
	return entries[i].Path < entries[j].Path
}
func (entries EnumerateEntries) Swap(i, j int) {
	entries[i], entries[j] = entries[j], entries[i]
}

type Core struct {
	bus USBBus

	normalSessions map[string]*session
	debugSessions  map[string]*session
	sessionsMutex  sync.Mutex // for atomic access to sessions

	allowStealing bool
	reset         bool

	callsInProgress int        // we cannot make calls and enumeration at the same time
	callMutex       sync.Mutex // for atomic access to callInProgress, plus prevent enumeration
	lastInfos       []USBInfo  // when call is in progress, use saved info for enumerating

	log *memorywriter.MemoryWriter
}

var (
	ErrWrongPrevSession = errors.New("wrong previous session")
	ErrSessionNotFound  = errors.New("session not found")
	ErrMalformedData    = errors.New("malformed data")
	ErrOtherCall        = errors.New("other call in progress")
)

const (
	VendorT1            = 0x534c
	ProductT1Firmware   = 0x0001
	VendorT2            = 0x1209
	ProductT2Bootloader = 0x53C0
	ProductT2Firmware   = 0x53C1
)

func New(bus USBBus, log *memorywriter.MemoryWriter, allowStealing, reset bool) *Core {
	c := &Core{
		bus:            bus,
		normalSessions: make(map[string]*session),
		debugSessions:  make(map[string]*session),
		log:            log,
		allowStealing:  allowStealing,
		reset:          reset,
	}
	return c
}

func (c *Core) Log(s string) {
	c.log.Println("core - " + s)
}

func (c *Core) Enumerate() ([]EnumerateEntry, error) {
	// Lock for atomic access to s.sessions.
	c.Log("enumerate locking sessionsMutex")
	c.sessionsMutex.Lock()
	defer c.sessionsMutex.Unlock()

	c.Log("enumerate locking callMutex")
	// Lock for atomic access to s.callInProgress.  It needs to be over
	// whole function, so that call does not actually start while
	// enumerating.
	c.callMutex.Lock()
	defer c.callMutex.Unlock()

	// Use saved info if call is in progress, otherwise enumerate.
	infos := c.lastInfos

	c.Log(fmt.Sprintf("enumerate callsInProgress %d", c.callsInProgress))
	if c.callsInProgress == 0 {
		c.Log("enumerate bus")
		busInfos, err := c.bus.Enumerate()
		if err != nil {
			return nil, err
		}
		infos = busInfos
		c.lastInfos = infos
	}

	entries := c.createEnumerateEntries(infos)
	c.Log("enumerate release disconnected")
	c.releaseDisconnected(infos, false)
	c.releaseDisconnected(infos, true)
	return entries, nil
}

func (c *Core) findSession(e *EnumerateEntry, path string, debug bool) {
	for _, ss := range c.sessions(debug) {
		if ss.path == path {
			// Copying to prevent overwriting on Acquire and
			// wrong comparison in Listen.
			ssidCopy := ss.id
			if debug {
				e.DebugSession = &ssidCopy
			} else {
				e.Session = &ssidCopy
			}
		}
	}
}

func (c *Core) createEnumerateEntry(info USBInfo) EnumerateEntry {
	e := EnumerateEntry{
		Path:    info.Path,
		Vendor:  info.VendorID,
		Product: info.ProductID,
		Type:    info.Type,
		Debug:   info.Debug,
	}
	c.findSession(&e, info.Path, false)
	c.findSession(&e, info.Path, true)
	return e
}

func (c *Core) createEnumerateEntries(infos []USBInfo) EnumerateEntries {
	entries := make(EnumerateEntries, 0, len(infos))
	for _, info := range infos {
		e := c.createEnumerateEntry(info)
		entries = append(entries, e)
	}
	entries.Sort()
	return entries
}

func (entries EnumerateEntries) Sort() {
	sort.Sort(entries)
}

func (c *Core) sessions(debug bool) map[string]*session {
	sessions := c.normalSessions
	if debug {
		sessions = c.debugSessions
	}
	return sessions
}

func (c *Core) releaseDisconnected(infos []USBInfo, debug bool) {

	for ssid, ss := range c.sessions(debug) {
		connected := false
		for _, info := range infos {
			if ss.path == info.Path {
				connected = true
			}
		}
		if !connected {
			c.Log(fmt.Sprintf("releasing disconnected device %s", ssid))
			err := c.release(ssid, true, debug)
			// just log if there is an error
			// they are disconnected anyway
			if err != nil {
				c.Log(fmt.Sprintf("Error on releasing disconnected device: %s", err))
			}
		}
	}
}

func (c *Core) Release(session string, debug bool) error {
	return c.release(session, false, debug)
}

func (c *Core) release(
	session string,
	disconnected bool,
	debug bool,
) error {
	c.Log(fmt.Sprintf("inner release - session %s", session))
	acquired := (c.sessions(debug))[session]
	if acquired == nil {
		c.Log("inner release - session not found")
		return ErrSessionNotFound
	}
	delete(c.sessions(debug), session)

	c.Log("inner release - bus close")
	err := acquired.dev.Close(disconnected)
	return err
}

func (c *Core) Listen(entries []EnumerateEntry, closeNotify <-chan bool) ([]EnumerateEntry, error) {
	c.Log("listen starting")

	const (
		iterMax   = 600
		iterDelay = 500 // ms
	)

	EnumerateEntries(entries).Sort()

	for i := 0; i < iterMax; i++ {
		c.Log("listen before enumerating")
		e, enumErr := c.Enumerate()
		if enumErr != nil {
			return nil, enumErr
		}
		for i := range e {
			e[i].Type = 0 // type is not exported/imported to json
		}
		if reflect.DeepEqual(entries, e) {
			c.Log("listen equal, waiting")
			select {
			case <-closeNotify:
				c.Log("listen request closed")
				return nil, nil
			default:
				time.Sleep(iterDelay * time.Millisecond)
			}
		} else {
			c.Log("listen different")
			entries = e
			break
		}
	}
	c.Log("listen encoding and exiting")
	return entries, nil
}

func (c *Core) findPrevSession(path string, debug bool) string {
	// note - sessionsMutex must be locked before entering this
	for _, ss := range c.sessions(debug) {
		if ss.path == path {
			return ss.id
		}
	}

	return ""
}

func (c *Core) Acquire(
	path, prev string,
	debug bool,
) (string, error) {

	c.Log("acquire - locking sessionsMutex")
	c.sessionsMutex.Lock()
	defer c.sessionsMutex.Unlock()

	c.Log(fmt.Sprintf("acquire - input path %s prev %s", path, prev))

	prevSession := c.findPrevSession(path, debug)

	c.Log(fmt.Sprintf("acquire - actually previous %s", prevSession))

	if prevSession != prev {
		return "", ErrWrongPrevSession
	}

	if (!c.allowStealing) && prevSession != "" {
		return "", ErrOtherCall
	}

	if prev != "" {
		c.Log("acquire - releasing previous")
		err := c.release(prev, false, debug)
		if err != nil {
			return "", err
		}
	}

	// reset device ONLY if no call on the other port
	// otherwise, USB reset stops other call
	otherSession := c.findPrevSession(path, !debug)
	reset := otherSession == "" && c.reset

	c.Log("acquire - trying to connect")
	dev, err := c.tryConnect(path, debug, reset)
	if err != nil {
		return "", err
	}

	id := c.newSession()
	if debug {
		id = "debug" + id
	}

	sess := &session{
		path: path,
		dev:  dev,
		call: 0,
		id:   id,
	}

	c.Log(fmt.Sprintf("acquire - new session is %s", id))

	(c.sessions(debug))[id] = sess

	return id, nil
}

// Chrome tries to read from trezor immediately after connecting,
// ans so do we.  Bad timing can produce error on s.bus.Connect.
// Try 3 times with a 100ms delay.
func (c *Core) tryConnect(path string, debug bool, reset bool) (USBDevice, error) {
	tries := 0
	for {
		c.Log(fmt.Sprintf("tryConnect - try number %d", tries))
		dev, err := c.bus.Connect(path, debug, reset)
		if err != nil {
			if tries < 3 {
				c.Log("tryConnect - sleeping")
				tries++
				time.Sleep(100 * time.Millisecond)
			} else {
				c.Log("tryConnect - too many times, exiting")
				return nil, err
			}
		} else {
			return dev, nil
		}
	}
}

var latestSessionID = 0

func (c *Core) newSession() string {
	latestSessionID++
	return strconv.Itoa(latestSessionID)
}

type CallMode int

const (
	CallModeRead      CallMode = 0
	CallModeWrite     CallMode = 1
	CallModeReadWrite CallMode = 2
)

func (c *Core) Call(
	body []byte,
	session string,
	mode CallMode,
	debug bool,
	closeNotify <-chan bool,
) ([]byte, error) {
	c.Log("call - start")

	c.Log("call - callMutex lock")
	c.callMutex.Lock()

	c.Log("call - callMutex set callInProgress true, unlock")
	c.callsInProgress++

	c.callMutex.Unlock()
	c.Log("call - callMutex unlock done")

	defer func() {
		c.Log("call - callMutex closing lock")
		c.callMutex.Lock()

		c.Log("call - callMutex set callInProgress false, unlock")
		c.callsInProgress--

		c.callMutex.Unlock()
		c.Log("call - callMutex closing unlock")
	}()

	c.Log("call - sessionsMutex lock")
	c.sessionsMutex.Lock()
	acquired := (c.sessions(debug))[session]
	c.sessionsMutex.Unlock()
	c.Log("call - sessionsMutex unlock done")

	if acquired == nil {
		return nil, ErrSessionNotFound
	}

	c.Log("call - checking other call on same session")
	freeToCall := atomic.CompareAndSwapInt32(&acquired.call, 0, 1)
	if !freeToCall {
		return nil, ErrOtherCall
	}

	c.Log("call - checking other call on same session done")
	defer func() {
		atomic.StoreInt32(&acquired.call, 0)
	}()

	finished := make(chan bool, 1)
	defer func() {
		finished <- true
	}()

	go func() {
		select {
		case <-finished:
			return
		case <-closeNotify:
			c.Log("call - detected request close, auto-release")
			errRelease := c.release(session, false, debug)
			if errRelease != nil {
				// just log, since request is already closed
				c.Log(fmt.Sprintf("Error while releasing: %s", errRelease.Error()))
			}
		}
	}()

	c.Log("call - before actual logic")
	bytes, err := c.readWriteDev(body, acquired.dev, mode)
	c.Log("call - after actual logic")

	return bytes, err
}

func (c *Core) writeDev(body []byte, device io.Writer) error {
	c.Log("readWrite - decodeRaw")
	msg, err := c.decodeRaw(body)
	if err != nil {
		return err
	}

	c.Log("readWrite - writeTo")
	_, err = msg.WriteTo(device)
	return err
}

func (c *Core) readDev(device io.Reader) ([]byte, error) {
	c.Log("readWrite - readFrom")
	msg, err := wire.ReadFrom(device, c.log)
	if err != nil {
		return nil, err
	}

	c.Log("readWrite - encoding back")
	return c.encodeRaw(msg)
}

func (c *Core) readWriteDev(
	body []byte,
	device io.ReadWriter,
	mode CallMode,
) ([]byte, error) {

	if mode == CallModeRead {
		if len(body) != 0 {
			return nil, errors.New("Non-empty body on read mode")
		}
		c.Log("readWrite - skipping write")
	} else {
		err := c.writeDev(body, device)
		if err != nil {
			return nil, err
		}
	}

	if mode == CallModeWrite {
		c.Log("readWrite - skipping read")
		return []byte{0}, nil
	}
	return c.readDev(device)
}

func (c *Core) decodeRaw(body []byte) (*wire.Message, error) {
	c.Log("decode - readAll")

	c.Log("decode - decodeString")

	if len(body) < 6 {
		c.Log("decode - body too short")
		return nil, ErrMalformedData
	}

	kind := binary.BigEndian.Uint16(body[0:2])
	size := binary.BigEndian.Uint32(body[2:6])
	data := body[6:]
	if uint32(len(data)) != size {
		c.Log("decode - wrong data length")
		return nil, ErrMalformedData
	}

	if wire.Validate(data) != nil {
		c.Log("decode - invalid data")
		return nil, ErrMalformedData
	}

	c.Log("decode - returning")
	return &wire.Message{
		Kind: kind,
		Data: data,

		Log: c.log,
	}, nil
}

func (c *Core) encodeRaw(msg *wire.Message) ([]byte, error) {
	c.Log("encode - start")
	var header [6]byte
	data := msg.Data
	kind := msg.Kind
	size := uint32(len(msg.Data))

	binary.BigEndian.PutUint16(header[0:2], kind)
	binary.BigEndian.PutUint32(header[2:6], size)

	res := append(header[:], data...)

	return res, nil
}
