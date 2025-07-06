package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/mdlayher/wol"
)

const (
	lport = "8080"

	raw string = "raw"
	udp string = "udp"
)

var mode string

func parseMac(r *http.Request) (net.HardwareAddr, error) {
	macParam := r.URL.Query().Get("mac")
	if macParam == "" {
		return nil, errors.New("missing 'mac' query parameter")
	}
	macAddr, err := net.ParseMAC(macParam)
	if err != nil {
		return nil, fmt.Errorf("invalid mac parametre: %v", err)
	}

	return macAddr, nil
}

func parseIf(r *http.Request) (*net.Interface, error) {
	ifcName := r.URL.Query().Get("if")
	if ifcName == "" {
		return nil, errors.New("request received without 'if' parameter")
	}

	ifc, err := net.InterfaceByName(ifcName)
	if err != nil {
		return nil, fmt.Errorf("could not find interface by name: %v", err)
	}

	return ifc, nil
}

func parseIp(r *http.Request) (*net.IP, error) {
	ipAddr := r.URL.Query().Get("ip")
	if ipAddr == "" {
		return nil, errors.New("request received without 'ip' parameter")
	}

	ip := net.ParseIP(ipAddr)
	if ip == nil {
		return nil, fmt.Errorf("wrong IP format")
	}
	return &ip, nil
}

func udpHandler() func(w http.ResponseWriter, r *http.Request) {
	wolCli, err := wol.NewClient()
	if err != nil {
		log.Fatalf("could not create wol client: %v", err)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hwAddr, err := parseMac(r)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, fmt.Sprintf("Error parsing mac address: %v", err), http.StatusBadRequest)
		}

		ipAddr, err := parseIp(r)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, fmt.Sprintf("Error parsing ip address: %v", err), http.StatusBadRequest)
		}

		err = wolCli.Wake(ipAddr.String(), hwAddr)
		if err != nil {
			log.Printf("Failed to send WOL packet to IP %s for MAC %s: %v", ipAddr, hwAddr.String(), err)
			http.Error(w, fmt.Sprintf("Failed to send WOL packet: %v", err), http.StatusInternalServerError)
			return
		}

		log.Printf("Successfully sent WOL packet to IP %s for MAC %s", ipAddr, hwAddr.String())
		fmt.Fprintf(w, "Magic packet successfully sentIP %s for MAC %s\n", ipAddr, hwAddr.String())
	})
}

func rawHandler() func(w http.ResponseWriter, r *http.Request) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ifc, err := parseIf(r)
		if err != nil {
			log.Printf("Failed to fetch interface from request : %v", err)
			http.Error(w, fmt.Sprintf("Failed to fetch interface from request : %v", err), http.StatusInternalServerError)
			return
		}

		wolCli, err := wol.NewRawClient(ifc)
		if err != nil {
			log.Fatalf("could not create raw wol client: %v", err)
		}
		defer wolCli.Close()

		hwAddr, err := parseMac(r)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, fmt.Sprintf("Error parsing mac address: %v", err), http.StatusBadRequest)
		}

		err = wolCli.Wake(hwAddr)
		if err != nil {
			log.Printf("Failed to send WOL packet from ifc %s for MAC %s: %v", ifc.Name, hwAddr.String(), err)
			http.Error(w, fmt.Sprintf("Failed to send WOL packet: %v", err), http.StatusInternalServerError)
			return
		}

		log.Printf("Successfully sent WOL packet from ifc %s for MAC %s", ifc.Name, hwAddr.String())
		fmt.Fprintf(w, "Magic packet successfully sent from ifc %s for MAC %s", ifc.Name, hwAddr.String())
	})
}

func main() {
	flag.StringVar(&mode, "mode", udp, "allows using udp socket or raw socket")
	flag.Parse()

	var httpHandlerFunc http.HandlerFunc
	switch mode {
	case raw:
		httpHandlerFunc = rawHandler()
	case udp:
		httpHandlerFunc = udpHandler()
	}
	http.HandleFunc("/wol", httpHandlerFunc)

	log.Printf("Starting Wake-on-LAN server mode: %s on port %s...", mode, lport)

	err := http.ListenAndServe(":"+lport, nil)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
