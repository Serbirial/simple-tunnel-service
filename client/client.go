package client

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"os"
	"time"
)

type TunnelConfig struct {
	ServiceName  string `json:"service_name"`
	PublicPort   int    `json:"public_port"`
	LocalAddress string `json:"local_address"`
	Secret       string `json:"secret"`
}

type ClientConfig struct {
	ServerAddress string         `json:"server_address"`
	Services      []TunnelConfig `json:"services"`
}

type AuthRequest struct {
	Secret     string `json:"secret"`
	Service    string `json:"service"`
	PublicPort int    `json:"public_port"`
}

func main() {
	configBytes, err := os.ReadFile("client.json")
	if err != nil {
		log.Fatalln("Missing client.json:", err)
	}
	var cfg ClientConfig
	if err := json.Unmarshal(configBytes, &cfg); err != nil {
		log.Fatalln("Invalid client.json:", err)
	}

	for _, svc := range cfg.Services {
		go startTunnel(cfg.ServerAddress, svc)
	}

	select {} // keep running
}

func startTunnel(serverAddr string, svc TunnelConfig) {
	for {
		conn, err := net.Dial("tcp", serverAddr)
		if err != nil {
			log.Printf("[%s] Connect error: %v", svc.ServiceName, err)
			time.Sleep(5 * time.Second)
			continue
		}

		auth := AuthRequest{
			Secret:     svc.Secret,
			Service:    svc.ServiceName,
			PublicPort: svc.PublicPort,
		}
		if err := json.NewEncoder(conn).Encode(auth); err != nil {
			log.Printf("[%s] Auth send error: %v", svc.ServiceName, err)
			conn.Close()
			time.Sleep(5 * time.Second)
			continue
		}

		log.Printf("[%s] Tunnel up (public :%d → %s)", svc.ServiceName, svc.PublicPort, svc.LocalAddress)
		go forwardTraffic(conn, svc.LocalAddress)
		io.Copy(io.Discard, conn)
		log.Printf("[%s] Disconnected, retrying...", svc.ServiceName)
		time.Sleep(3 * time.Second)
	}
}

func forwardTraffic(serverConn net.Conn, localAddr string) {
	for {
		localConn, err := net.Dial("tcp", localAddr)
		if err != nil {
			log.Printf("Local connect error (%s): %v", localAddr, err)
			time.Sleep(2 * time.Second)
			continue
		}
		go io.Copy(localConn, serverConn)
		io.Copy(serverConn, localConn)
		localConn.Close()
	}
}
