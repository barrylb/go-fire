package main

/*
GoFire HTTP server for controlling Mertik Maxitrol GV60 via Raspberry Pi with relay board.

Supported operations:
  Turn on: http://127.0.0.1:8600/on
  Turn off: http://127.0.0.1:8600/off
  Flame up: http://127.0.0.1:8600/flameup
  Flame down: http://127.0.0.1:8600/flamedown

Mertik Maxitrol GV60 documentation:
http://www.ortalglobal.com/wp-content/uploads/2018/08/External-Source-Operation-Wall-Switch-Wiring-Diagram.pdf

3 Channel Relay for Raspberry Pi:
https://www.waveshare.com/wiki/RPi_Relay_Board

Channels on the relay board should be wired to the corresponding contact number on the GV60.

*/

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/warthog618/gpiod"
	"github.com/warthog618/gpiod/device/rpi"
	"golang.org/x/sync/semaphore"
)

var sem = semaphore.NewWeighted(1)
var chip *gpiod.Chip
var ch1 *gpiod.Line
var ch2 *gpiod.Line
var ch3 *gpiod.Line

func offHandler(w http.ResponseWriter, r *http.Request) {
	// OFF: close contacts 1 & 2 & 3 for 1 second
	if sem.TryAcquire(1) {
		defer sem.Release(1)
		ch1.SetValue(0)
		ch2.SetValue(0)
		ch3.SetValue(0)
		time.Sleep(1 * time.Second)
		ch1.SetValue(1)
		ch2.SetValue(1)
		ch3.SetValue(1)
		fmt.Fprintf(w, "off_ok")
	} else {
		fmt.Fprintf(w, "off_busy")
	}
}

func onHandler(w http.ResponseWriter, r *http.Request) {
	// ON (Ignition): close contacts 1 & 3 for 1 second
	if sem.TryAcquire(1) {
		defer sem.Release(1)
		ch1.SetValue(0)
		ch2.SetValue(1)
		ch3.SetValue(0)
		time.Sleep(1 * time.Second)
		ch1.SetValue(1)
		ch3.SetValue(1)
		fmt.Fprintf(w, "on_ok")
	} else {
		fmt.Fprintf(w, "on_busy")
	}
}

func flameUpHandler(w http.ResponseWriter, r *http.Request) {
	// FLAME UP: close contact 1 (up to 12 seconds from min flame to full flame; let's do it in 2 sec increments)
	if sem.TryAcquire(1) {
		defer sem.Release(1)
		ch1.SetValue(0)
		ch2.SetValue(1)
		ch3.SetValue(1)
		time.Sleep(2 * time.Second)
		ch1.SetValue(1)
		fmt.Fprintf(w, "flameup_ok")
	} else {
		fmt.Fprintf(w, "flameup_busy")
	}
}

func flameDownHandler(w http.ResponseWriter, r *http.Request) {
	// FLAME DOWN: close contact 3 (up to 12 seconds from full flame down to min flame; let's do it in 2 sec increments)
	if sem.TryAcquire(1) {
		defer sem.Release(1)
		ch1.SetValue(1)
		ch2.SetValue(1)
		ch3.SetValue(0)
		time.Sleep(2 * time.Second)
		ch3.SetValue(1)
		fmt.Fprintf(w, "flamedown_ok")
	} else {
		fmt.Fprintf(w, "flamedown_busy")
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to GoFire server. Supported handlers: /off /on /flameup /flamedown")
}

func main() {
	var err error
	if chip, err = gpiod.NewChip("gpiochip0"); err != nil {
		panic(err)
	}
	defer chip.Close()
	// Setup the three relay channels using GPIO lines defined by https://www.waveshare.com/wiki/RPi_Relay_Board
	if ch1, err = chip.RequestLine(rpi.GPIO26, gpiod.AsOutput(1)); err != nil {
		panic(err)
	}
	if ch2, err = chip.RequestLine(rpi.GPIO20, gpiod.AsOutput(1)); err != nil {
		panic(err)
	}
	if ch3, err = chip.RequestLine(rpi.GPIO21, gpiod.AsOutput(1)); err != nil {
		panic(err)
	}
	//
	var listenAddr string
	flag.StringVar(&listenAddr, "listen_on", ":8600", "Listen address; default :8600")
	flag.Parse()
	//
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/off", offHandler)
	http.HandleFunc("/on", onHandler)
	http.HandleFunc("/flameup", flameUpHandler)
	http.HandleFunc("/flamedown", flameDownHandler)
	fmt.Printf("GoFire server listening on %v\n", listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}
