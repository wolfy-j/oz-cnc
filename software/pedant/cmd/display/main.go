package main

import (
	"flag"
	"github.com/fatih/color"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"
)

func updateStatus(d *oz.Display, status string) {
	switch status {
	case "paused":
		status = "pause"
		break
	case "running":
		status = "run"
		break
	}

	if strings.Trim(status, "") == "" {
		status = "BOOT"
	}

	d.Status(strings.ToUpper(status))
}

func main() {
	var (
		host     = flag.String("h", "localhost:8000", "CNC.js host")
		username = flag.String("u", "oz", "CNC.js username")
		password = flag.String("p", "", "CNC.js password")
	)

	display, err := oz.NewDisplay()
	if err != nil {
		log.Panic(err)
	}
	defer display.Close()

	display.Splash("booting server")
	cncjs := oz.NewClient(host)

	display.Splash("connecting")
	err = cncjs.SignIn(username, password)
	if err != nil {
		display.Splash("error connecting")
		log.Panic(err)
	}

	display.Splash("connected")
	time.Sleep(time.Second * 1)

	display.Splash("waiting for server")
	time.Sleep(time.Second * 2)

	display.Splash("connecting")
	err = cncjs.Connect()
	if err != nil {
		log.Panic(err)
	}
	defer cncjs.Close()

	// update handlers
	cncjs.OnState = func(state string) error {
		log.Println(color.GreenString(state))

		updateStatus(display, state)

		if state == "Idle" {
			display.Progress(0)
			display.ETA("")
			display.Message("OK: 100.70.0.38") // todo: get IP dynamically
		}

		display.Refresh()

		return nil
	}
	cncjs.OnGcode = func(name, gcode string) error {
		log.Println("gcode loaded:", color.BlueString(name))
		display.Message(name)
		return nil
	}
	cncjs.OnStatus = func(status oz.SenderStatus) error {
		duration := time.Duration(status.RemainingTime * 1000000)
		duration = duration.Round(time.Second)

		if duration == 0 {
			display.Progress(0)
			display.ETA("")
			// nope
			return nil
		}

		log.Printf(
			"remaining: "+color.YellowString("%s")+", progress: %v",
			duration,
			(100 * status.Received / status.Total),
		)

		if status.Hold {
			updateStatus(display, "hold")
		}

		display.Blue(status.Hold)
		display.Progress(byte(100 * status.Received / status.Total))
		display.ETA(duration.String())
		display.DisplayMode(1)
		display.Refresh()

		return nil
	}
	cncjs.OnGrlb = func(grbl oz.GrblState) error {
		log.Printf(
			color.HiBlackString("feed: %v, coord: [%v, %v, %v]"),
			grbl.Status.Feedrate,
			grbl.Status.Wpos.X,
			grbl.Status.Wpos.Y,
			grbl.Status.Wpos.Z,
		)

		if grbl.Status.Wpos.X[0] != '-' {
			grbl.Status.Wpos.X = " " + grbl.Status.Wpos.X
		}

		if grbl.Status.Wpos.Y[0] != '-' {
			grbl.Status.Wpos.Y = " " + grbl.Status.Wpos.Y
		}

		if grbl.Status.Wpos.Z[0] != '-' {
			grbl.Status.Wpos.Z = " " + grbl.Status.Wpos.Z
		}

		display.X(grbl.Status.Wpos.X)
		display.Y(grbl.Status.Wpos.Y)
		display.Z(grbl.Status.Wpos.Z)

		updateStatus(display, grbl.Status.ActiveState)

		display.Feedrate(strconv.Itoa(grbl.Status.Feedrate))
		display.DisplayMode(1)
		display.Refresh()

		return nil
	}
	cncjs.OnSerial = func(msg string) error {
		msg = strings.Replace(msg, "MSG:", "", -1)
		msg = strings.Replace(msg, "GCode Comment...", "", -1)
		msg = strings.Replace(msg, "Change to", "", -1)

		msg = strings.Trim(msg, "[] ")
		log.Println(color.MagentaString(msg))

		display.Message(msg)
		return nil
	}

	display.Splash("OK: 100.70.0.38") // todo: get IP dynamically
	display.Green(true)

	go func() {
		// keep display alive
		t := time.NewTicker(time.Millisecond * 500)
		for {
			select {
			case <-t.C:
				display.SilentRefresh()
			}
		}
	}()

	err = cncjs.Start("/dev/ttyS3")
	if err != nil {
		log.Panic(err)
	}

	// Setup a channel to receive a signal
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt)

	for _ = range done {
		os.Exit(0)
	}
}
