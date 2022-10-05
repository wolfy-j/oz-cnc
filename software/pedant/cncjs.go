package pedant

import (
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"github.com/gorilla/websocket"
	"log"
	"time"
)

// payload formats
type GrblState struct {
	Status struct {
		ActiveState string `json:"activeState"`
		Mpos        struct {
			X string `json:"x"`
			Y string `json:"y"`
			Z string `json:"z"`
		} `json:"mpos"`
		Wpos struct {
			X string `json:"x"`
			Y string `json:"y"`
			Z string `json:"z"`
		} `json:"wpos"`
		Ov       []int `json:"ov"`
		SubState int   `json:"subState"`
		Feedrate int   `json:"feedrate"`
		Spindle  int   `json:"spindle"`
		Wco      struct {
			X string `json:"x"`
			Y string `json:"y"`
			Z string `json:"z"`
		} `json:"wco"`
	} `json:"status"`
	Parserstate struct {
		Modal struct {
			Motion   string `json:"motion"`
			Wcs      string `json:"wcs"`
			Plane    string `json:"plane"`
			Units    string `json:"units"`
			Distance string `json:"distance"`
			Feedrate string `json:"feedrate"`
			Spindle  string `json:"spindle"`
			Coolant  string `json:"coolant"`
		} `json:"modal"`
		Tool     string `json:"tool"`
		Feedrate string `json:"feedrate"`
		Spindle  string `json:"spindle"`
	} `json:"parserstate"`
}
type SenderStatus struct {
	Sp         int         `json:"sp"`
	Hold       bool        `json:"hold"`
	HoldReason interface{} `json:"holdReason"`
	Name       string      `json:"name"`
	Context    struct {
		Global struct {
		} `json:"global"`
		Xmin  int     `json:"xmin"`
		Xmax  int     `json:"xmax"`
		Ymin  int     `json:"ymin"`
		Ymax  int     `json:"ymax"`
		Zmin  int     `json:"zmin"`
		Zmax  int     `json:"zmax"`
		Mposx float64 `json:"mposx"`
		Mposy float64 `json:"mposy"`
		Mposz float64 `json:"mposz"`
		Mposa int     `json:"mposa"`
		Mposb int     `json:"mposb"`
		Mposc int     `json:"mposc"`
		Posx  float64 `json:"posx"`
		Posy  float64 `json:"posy"`
		Posz  float64 `json:"posz"`
		Posa  int     `json:"posa"`
		Posb  int     `json:"posb"`
		Posc  int     `json:"posc"`
		Modal struct {
			Motion   string `json:"motion"`
			Wcs      string `json:"wcs"`
			Plane    string `json:"plane"`
			Units    string `json:"units"`
			Distance string `json:"distance"`
			Feedrate string `json:"feedrate"`
			Spindle  string `json:"spindle"`
			Coolant  string `json:"coolant"`
		} `json:"modal"`
		Tool   int `json:"tool"`
		Params struct {
		} `json:"params"`
		Math struct {
		} `json:"Math"`
		JSON struct {
		} `json:"JSON"`
	} `json:"context"`
	Size          int     `json:"size"`
	Total         int     `json:"total"`
	Sent          int     `json:"sent"`
	Received      int     `json:"received"`
	StartTime     int64   `json:"startTime"`
	FinishTime    int     `json:"finishTime"`
	ElapsedTime   int     `json:"elapsedTime"`
	RemainingTime float64 `json:"remainingTime"`
}

type OnGrlb func(GrblState) error
type OnStatus func(SenderStatus) error
type OnGcode func(name, gcode string) error
type OnState func(state string) error
type OnSerial func(state string) error

type Client struct {
	host       string
	token, sid string
	ws         *websocket.Conn
	OnGrlb     OnGrlb
	OnStatus   OnStatus
	OnGcode    OnGcode
	OnState    OnState
	OnSerial   OnSerial
}

func NewClient(host string) *Client {
	return &Client{host: host}
}

func (c *Client) SignIn(name, password string) error {
	count := 0
	for {
		auth := &struct {
			Enabled bool   `json:"enabled"`
			Token   string `json:"token"`
			Name    string `json:"name"`
		}{}

		err := PostJson(
			fmt.Sprintf("http://%s/api/signin", c.host),
			struct {
				Name     string `json:"name"`
				Password string `json:"password"`
			}{Name: name, Password: password},
			&auth,
		)

		if err == nil {
			c.token = auth.Token
			return nil
		}

		count++
		if count > 100 {
			return err
		}

		time.Sleep(time.Second)
	}
}

func (c *Client) Connect() error {
	// getting sid
	sidData, err := GetRaw(
		fmt.Sprintf("http://%s/socket.io/?token=%s&EIO=3&transport=polling&t=ODK-4e_", c.host, c.token),
	)

	if err != nil {
		return err
	}

	sid := &struct {
		SID string `json:"sid"`
	}{}

	err = json.Unmarshal(sidData[4:], &sid)
	if err != nil {
		return err
	}

	c.sid = sid.SID

	// ws
	c.ws, _, err = websocket.DefaultDialer.Dial(
		fmt.Sprintf(
			"ws://%s/socket.io/?token=%s&EIO=3&transport=websocket&sid=%s",
			c.host,
			c.token,
			c.sid,
		),
		nil,
	)

	return err
}

func (c *Client) Start(port string) error {
	go func() {
		for {
			_, message, err := c.ws.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}

			err = c.handleMessage(message)
			if err != nil {
				log.Println("handle:", err)
			}
		}
	}()

	// protocol
	err := c.ws.WriteMessage(websocket.TextMessage, []byte("5"))
	if err != nil {
		return err
	}

	// todo: wait response

	//420["open","/dev/ttyS3",{"controllerType":"Grbl","baudrate":115200,"rtscts":false}]
	err = c.ws.WriteMessage(websocket.TextMessage, []byte("42[\"open\", \""+port+"\",{\"controllerType\":\"Grbl\",\"baudrate\":115200,\"rtscts\":false}]"))
	if err != nil {
		return err
	}

	// todo: wait response

	//42["command","/dev/ttyS3","reset"]
	err = c.ws.WriteMessage(websocket.TextMessage, []byte("42[\"command\", \""+port+"\",\"reset\"]"))
	if err != nil {
		return err
	}

	// todo: wait response

	// to avoid timeouts
	t := time.NewTicker(time.Second * 5)
	for {
		select {
		case <-t.C:
			err = c.ws.WriteMessage(websocket.TextMessage, []byte("2"))
			if err != nil {
				return err
			}
		}
	}
}

func (c *Client) handleMessage(msg []byte) error {
	if string(msg) == "3" {
		// pong
		return nil
	}

	prefix := string(msg[0:3])
	if prefix != "42[" {
		log.Printf("got: %s", color.YellowString(string(msg)))
		// not a payload
		return nil
	}

	msg = msg[2:]

	// parsing
	payload := make([]json.RawMessage, 0)
	err := json.Unmarshal(msg, &payload)
	if err != nil {
		return err
	}

	var topic string
	err = json.Unmarshal(payload[0], &topic)
	if err != nil {
		return err
	}

	return c.handle(topic, payload[1:])
}

func (c *Client) handle(topic string, data []json.RawMessage) error {
	str := make([]string, len(data))
	for i, m := range data {
		str[i] = string(m)
	}

	var err error
	switch topic {
	case "controller:state",
		"startup",
		"serialport:open",
		"serialport:write",
		"controller:settings",
		"Grbl:settings",
		"feeder:status":
		// supressed
		// todo: unsuppress??? controller:state?
		return nil
	case "workflow:state":
		var v string
		err = json.Unmarshal(data[0], &v)
		if err != nil {
			return err
		}

		if c.OnState != nil {
			return c.OnState(v)
		}
	case "serialport:read":
		var v string
		err = json.Unmarshal(data[0], &v)
		if err != nil {
			return err
		}

		if c.OnSerial != nil {
			return c.OnSerial(v)
		}
	case "gcode:load":
		var n string
		err = json.Unmarshal(data[0], &n)
		if err != nil {
			return err
		}

		var g string
		err = json.Unmarshal(data[1], &g)
		if err != nil {
			return err
		}

		if c.OnGcode != nil {
			return c.OnGcode(n, g)
		}
	case "sender:status":
		var s SenderStatus
		err = json.Unmarshal(data[0], &s)
		if err != nil {
			return err
		}

		if c.OnStatus != nil {
			return c.OnStatus(s)
		}
	case "Grbl:state":
		var s GrblState
		err = json.Unmarshal(data[0], &s)
		if err != nil {
			return err
		}

		if c.OnGrlb != nil {
			return c.OnGrlb(s)
		}
	default:
		log.Printf("%s: %v", color.YellowString(topic), str)
	}

	return nil
}

func (c *Client) Close() error {
	return c.ws.Close()
}
