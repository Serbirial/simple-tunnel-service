package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

type AuthRequest struct {
	Secret     string `json:"secret"`
	Service    string `json:"service"`
	PublicPort int    `json:"public_port"`
}

var (
	serviceSecrets map[string]string
	globalSecret   string
)

func loadServiceSecrets(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &serviceSecrets)
}

func loadGlobalSecret(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	globalSecret = string(data)
	return nil
}

func main() {
	if err := loadServiceSecrets("service_secrets.json"); err != nil {
		log.Println("No service_secrets.json found, using global secret only")
		serviceSecrets = make(map[string]string)
	}
	if err := loadGlobalSecret("secret.txt"); err != nil {
		log.Fatalln("Missing required secret.txt:", err)
	}

	log.Println("Server listening on :7000 for tunnel connections")
	go listenForClients(":7000")
	select {}
}

func listenForClients(addr string) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalln("Failed to listen on", addr, ":", err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Connection error:", err)
			continue
		}
		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	defer conn.Close()

	var req AuthRequest
	if err := json.NewDecoder(conn).Decode(&req); err != nil {
		log.Println("Invalid auth payload:", err)
		return
	}

	if expected, ok := serviceSecrets[req.Service]; ok {
		if req.Secret != expected {
			log.Printf("Wrong secret for service '%s'\n", req.Service)
			return
		}
	} else {
		if req.Secret != globalSecret {
			log.Printf("Unauthorized access to service '%s' with global secret\n", req.Service)
			return
		}
	}

	publicAddr := fmt.Sprintf(":%d", req.PublicPort)
	log.Printf("Accepted tunnel for service '%s' on %s\n", req.Service, publicAddr)

	publicLn, err := net.Listen("tcp", publicAddr)
	if err != nil {
		log.Println("Failed to bind to port", publicAddr, ":", err)
		return
	}
	defer publicLn.Close()

	for {
		pubConn, err := publicLn.Accept()
		if err != nil {
			log.Println("Public accept error:", err)
			return
		}
		go bridgeConnections(pubConn, conn)
	}
}

func bridgeConnections(pubConn, clientConn net.Conn) {
	defer pubConn.Close()
	sideA, sideB := net.Pipe()
	go io.Copy(sideA, pubConn)
	go io.Copy(pubConn, sideA)
	go io.Copy(clientConn, sideB)
	io.Copy(sideB, clientConn)
}
