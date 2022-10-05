package pedant

import (
	"bytes"
	"github.com/d2r2/go-i2c"
	"time"
)

const (
	DISPLAY_MODE    = 10
	REFRESH         = 1
	MESSAGE         = 2
	UPDATE_STATUS   = 3
	UPDATE_ETA      = 4
	UPDATE_PROGRESS = 5
	UPDATE_X        = 6
	UPDATE_Y        = 7
	UPDATE_Z        = 8
	UPDATE_FEEDRATE = 9
	UPDATE_RED      = 12
	UPDATE_GREEN    = 13
	UPDATE_BLUE     = 14
	UPDATE_BLINK    = 20
)

const Addr = 77

type Display struct {
	i2c *i2c.I2C
}

const delay = time.Millisecond

func NewDisplay() (*Display, error) {
	// Create new connection to I2C bus on 2 line with address 0x27
	ic, err := i2c.NewI2C(Addr, 0)
	if err != nil {
		return nil, err
	}

	return &Display{i2c: ic}, nil
}

func (m *Display) Message(msg string) error {
	if len(msg) > 16 {
		msg = msg[0:16]
	}

	return m.sendString(MESSAGE, msg)
}

func (m *Display) Splash(msg string) error {
	m.sendString(MESSAGE, msg)
	m.DisplayMode(0)
	return m.Refresh()
}

func (m *Display) Status(msg string) error {
	if msg == "" {
		msg = "BOOT"
	}
	return m.sendString(UPDATE_STATUS, msg)
}

func (m *Display) ETA(msg string) error {
	return m.sendString(UPDATE_ETA, msg)
}

func (m *Display) X(msg string) error {
	return m.sendString(UPDATE_X, msg)
}

func (m *Display) Y(msg string) error {
	return m.sendString(UPDATE_Y, msg)
}

func (m *Display) Z(msg string) error {
	return m.sendString(UPDATE_Z, msg)
}

func (m *Display) Feedrate(msg string) error {
	return m.sendString(UPDATE_FEEDRATE, msg)
}

func (m *Display) Progress(progress byte) error {
	return m.sendPayload(UPDATE_PROGRESS, []byte{progress})
}

func (m *Display) Blink(duration byte) error {
	return m.sendPayload(UPDATE_BLINK, []byte{duration})
}

func (m *Display) Red(on bool) error {
	var v byte
	if on {
		v = 1
	}

	return m.sendPayload(UPDATE_RED, []byte{v})
}

func (m *Display) Green(on bool) error {
	var v byte
	if on {
		v = 1
	}

	return m.sendPayload(UPDATE_GREEN, []byte{v})
}

func (m *Display) Blue(on bool) error {
	var v byte
	if on {
		v = 1
	}

	return m.sendPayload(UPDATE_BLUE, []byte{v})
}

func (m *Display) DisplayMode(mode byte) error {
	return m.sendPayload(DISPLAY_MODE, []byte{mode})
}

func (m *Display) Refresh() error {
	m.Blink(2)
	return m.sendPayload(REFRESH, []byte{})
}

func (m *Display) SilentRefresh() error {
	return m.sendPayload(REFRESH, []byte{})
}

func (m *Display) sendString(code byte, msg string) error {
	buf := bytes.NewBuffer(nil)
	buf.WriteByte(byte(len(msg) + 1))
	buf.WriteString(msg)
	buf.Write([]byte{0})

	return m.sendPayload(code, buf.Bytes())
}

func (m *Display) sendPayload(code byte, data []byte) error {
	buf := bytes.NewBuffer(nil)
	buf.Write([]byte{code})
	buf.Write(data)

	_, err := m.i2c.WriteBytes(buf.Bytes())
	if err != nil {
		return err
	}

	time.Sleep(delay)
	arr := make([]byte, 2)
	_, err = m.i2c.ReadBytes(arr)
	if err != nil {
		return err
	}

	return nil
}

func (m *Display) Close() error {
	return m.i2c.Close()
}
