package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
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

func getPortFromFile(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", fmt.Errorf("failed to open port file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		portStr := strings.TrimSpace(scanner.Text())
		portNum, err := strconv.Atoi(portStr)
		if err != nil || portNum < 1 || portNum > 65535 {
			return "", fmt.Errorf("invalid port number in file: %s", portStr)
		}

		// Check if port is free by trying to listen on it
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", portNum))
		if err != nil {
			return "", fmt.Errorf("port %d is already in use or unavailable", portNum)
		}
		ln.Close()

		return portStr, nil
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading port file: %w", err)
	}
	return "", fmt.Errorf("port file is empty")
}

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
	port, err := getPortFromFile("port.txt")
	if err != nil {
		log.Fatalf("Error reading port: %v", err)
	}
	log.Printf("Server listening on :%s for tunnel connections", port)
	go listenForClients(":" + port)
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
