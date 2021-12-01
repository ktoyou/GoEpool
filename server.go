package main

import (
	"flag"
	"log"
	"strconv"
	"syscall"
	"time"
)

type ServerConfig struct {
	Port           int
	MaxConnections int
	MaxQueue       int
	Address        string
}

func GetEpollFd(size int) int {
	epfd, err := syscall.EpollCreate(size)
	if err != nil {
		log.Fatal(err.Error())
	}
	return epfd
}

func GetSocketFd() int {
	sfd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		log.Fatal(err.Error())
	}
	return sfd
}

func OctetToByte(octet string) byte {
	octetnum, _ := strconv.Atoi(octet)
	return byte(octetnum)
}

func AddrToBytes(address string) [4]byte {
	var octet string
	var addr [4]byte
	var octetCount int
	for i := 0; i < len(address); i++ {
		if address[i] == '.' {
			addr[octetCount] = OctetToByte(octet)
			octetCount++
			octet = ""
			continue
		}
		octet += string(address[i])
		if octetCount == 3 {
			addr[3] = OctetToByte(octet)
		}
	}
	return addr
}

func Bind(port int, sfd int, address string) {
	var addr syscall.SockaddrInet4
	addr.Addr = AddrToBytes(address)
	addr.Port = port
	err := syscall.Bind(sfd, &addr)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func StartServer(sfd int, maxQueue int, address string) {
	Bind(8080, sfd, address)
	err := syscall.Listen(sfd, maxQueue)
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Print("server started")
}

func ListenEvents(epfd int, events []syscall.EpollEvent) {
	for true {
		if epsize, err := syscall.EpollWait(epfd, events, 0); err == nil {
			for i := 0; i < epsize; i++ {
				buffer := make([]byte, 1024)
				readed, err := syscall.Read(int(events[i].Fd), buffer)
				if err != nil {
					log.Fatal(err.Error())
				}
				log.Printf("readed %s bytes", strconv.Itoa(readed))
				log.Printf("message -> %s", string(buffer))
			}
		} else {
			log.Fatal(err.Error())
		}
		time.Sleep(10)
	}
}

func AddEvent(epfd int, cfd int) {
	ev := new(syscall.EpollEvent)
	ev.Events = syscall.EPOLLIN | syscall.EPOLLOUT | syscall.EPOLLONESHOT
	ev.Fd = int32(cfd)
	err := syscall.EpollCtl(epfd, syscall.EPOLL_CTL_ADD, cfd, ev)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func main() {
	var config ServerConfig

	flag.IntVar(&config.Port, "p", 8000, "Server port. Usage: -p 8080")
	flag.IntVar(&config.MaxConnections, "m", 16, "Max connections. Usage: -m 16")
	flag.IntVar(&config.MaxQueue, "q", 10, "Max queue. Usage: -q 10")
	flag.StringVar(&config.Address, "a", "127.0.0.1", "Address. Usage: -a 127.0.0.1")
	flag.Parse()

	AddrToBytes(config.Address)

	events := make([]syscall.EpollEvent, config.MaxConnections)
	epfd := GetEpollFd(config.MaxConnections)
	sfd := GetSocketFd()

	StartServer(sfd, config.MaxQueue, config.Address)
	go ListenEvents(epfd, events)

	for true {
		if cfd, _, err := syscall.Accept(sfd); err == nil {
			log.Print("client connected")
			AddEvent(epfd, cfd)
		} else {
			log.Fatal(err.Error())
		}
	}
}
