package main

////////////////////////////////////////////////////////////////////////////////

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"

	wol "github.com/sabhiram/go-wol"

	tb "gopkg.in/tucnak/telebot.v2"
)

// var (
// 	// Define holders for the cli arguments we wish to parse.
// 	cliFlags struct {
// 		Version            bool   `short:"v" long:"version"`
// 		Help               bool   `short:"h" long:"help"`
// 		BroadcastInterface string `short:"i" long:"interface" default:""`
// 		BroadcastIP        string `short:"b" long:"bcast" default:"255.255.255.255"`
// 		UDPPort            string `short:"p" long:"port" default:"9"`
// 	}
// )

type sender interface {
	Send(to tb.Recipient, what interface{}, options ...interface{}) (*tb.Message, error)
}

type LogToTelegram struct {
	Master string
	Bot    sender
}

func (l LogToTelegram) Recipient() string {
	return l.Master
}

func (l LogToTelegram) Write(p []byte) (int, error) {
	_, err := l.Bot.Send(l, string(p))
	return len(p), err
}

func (w WakeOnLan) IpFromInterface() (*net.UDPAddr, error) {
	ief, err := net.InterfaceByName(w.bcastInterface)
	if err != nil {
		return nil, err
	}

	addrs, err := ief.Addrs()
	if err == nil && len(addrs) <= 0 {
		err = fmt.Errorf("no address associated with interface %s", w.bcastInterface)
	}
	if err != nil {
		return nil, err
	}

	// Validate that one of the addrs is a valid network IP address.
	for _, addr := range addrs {
		switch ip := addr.(type) {
		case *net.IPNet:
			if !ip.IP.IsLoopback() && ip.IP.To4() != nil {
				return &net.UDPAddr{
					IP: ip.IP,
				}, nil
			}
		}
	}
	return nil, fmt.Errorf("no address associated with interface %s", w.bcastInterface)
}

type WakeOnLan struct {
	macAddr        string
	bcastInterface string
	BroadcastIP    string
	UDPPort        string
}

func NewWakeOnLan(macAddr string) *WakeOnLan {
	return &WakeOnLan{
		macAddr:        macAddr,
		bcastInterface: "",
		BroadcastIP:    "255.255.255.255",
		UDPPort:        "9",
	}
}

// Run the wake command.
func (w WakeOnLan) Wake() (err error) {
	// Populate the local address in the event that the broadcast interface has
	// been set.
	var localAddr *net.UDPAddr
	if w.bcastInterface != "" {
		localAddr, err = w.IpFromInterface()
		if err != nil {
			return err
		}
	}

	// The address to broadcast to is usually the default `255.255.255.255` but
	// can be overloaded by specifying an override in the CLI arguments.
	bcastAddr := fmt.Sprintf("%s:%s", w.BroadcastIP, w.UDPPort)
	udpAddr, err := net.ResolveUDPAddr("udp", bcastAddr)
	if err != nil {
		return err
	}

	// Build the magic packet.
	mp, err := wol.New(w.macAddr)
	if err != nil {
		return err
	}

	// Grab a stream of bytes to send.
	bs, err := mp.Marshal()
	if err != nil {
		return err
	}

	// Grab a UDP connection to send our packet of bytes.
	conn, err := net.DialUDP("udp", localAddr, udpAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	log.Printf("Attempting to send a magic packet to MAC %s\n", w.macAddr)
	log.Printf("... Broadcasting to: %s\n", bcastAddr)
	n, err := conn.Write(bs)
	if err == nil && n != 102 {
		err = fmt.Errorf("magic packet sent was %d bytes (expected 102 bytes sent)", n)
	}
	if err != nil {
		return err
	}

	log.Printf("Magic packet sent successfully to %s\n", w.macAddr)
	return nil
}

// Main entry point for binary.
func main() {

	mac := flag.String("mac", "", "Mac address to wake up")
	iface := flag.String("iface", "", "network interface to send the package")

	flag.Parse()

	token, ok := os.LookupEnv("BOT_TOKEN")
	if !ok {
		panic("BOT_TOKEN is not defined")
	}

	b, err := tb.NewBot(tb.Settings{
		Token:  token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})

	log.SetOutput(io.MultiWriter(
		LogToTelegram{
			Master: "142825882",
			Bot:    b,
		},
		os.Stdout,
	))

	if err != nil {
		log.Fatal(err)
		return
	}

	wol := NewWakeOnLan(*mac)
	wol.bcastInterface = *iface

	b.Handle("/on", func(m *tb.Message) {
		b.Send(m.Sender, "I'm waking up your PC")
		err := wol.Wake()
		if err != nil {
			b.Send(m.Sender, "Wops! something went wrong...")
			log.Println(err)
		}
	})
	b.Handle("/list", func(m *tb.Message) {
		b.Send(m.Sender, "Scanning for computers")
		ifaceIP, err := wol.IpFromInterface()
		if err != nil {
			log.Println(err)
		}
		log.Println(ifaceIP)
		ip := ifaceIP.IP.Mask(ifaceIP.IP.DefaultMask())
		out, err := NewNmapRun(ip.String())
		if err != nil {
			log.Println(err)
		}
		addresses := out.GetAddressList()
		log.Println(strings.Join(addresses, "\n"))
		b.Send(m.Sender, "These are the computers that I've found in local network \n"+strings.Join(addresses, "\n"))
	})

	log.Println("Bot ready")
	b.Start()
}
