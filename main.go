package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"io"
	"log"
	"net"
	"os"
	"sync"
)

func handleConnection(client net.Conn) {
	defer client.Close()

	log.Printf("handleConnection: got Client; dialing Docker sock")

	dockerSock, err := net.Dial("unix", dockerSockPath)
	if err != nil {
		log.Printf("handleConnection: Docker sock dial failed: %s", err.Error())
		return
	}
	defer dockerSock.Close()

	bidiPipe(client, "Client", dockerSock, "Docker")

	log.Printf("handleConnection: closing")
}

func mainInternal() error {
	serverCertKeyBase64 := os.Getenv("SERVERCERT_KEY")
	if serverCertKeyBase64 == "" {
		return errors.New("SERVERCERT_KEY not defined")
	}

	serverCertKey, err := base64.StdEncoding.DecodeString(serverCertKeyBase64)
	if err != nil {
		return err
	}

	serverCert, err := tls.X509KeyPair([]byte(serverCert), serverCertKey)
	if err != nil {
		log.Fatalf("server: loadkeys: %s", err)
	}

	tlsConfig := tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    getCaCert(),
	}

	log.Printf("Starting to listen on %s", addr)
	tcpTlsListener, err := tls.Listen("tcp", addr, &tlsConfig)

	for {
		conn, err := tcpTlsListener.Accept()
		if err != nil {
			return err
			// handle error
		}
		go handleConnection(conn)
	}

	return nil
}

func main() {
	if err := mainInternal(); err != nil {
		panic(err)
	}
}

func bidiPipe(party1 io.ReadWriteCloser, party1Name string, party2 io.ReadWriteCloser, party2Name string) {
	allIoFinished := &sync.WaitGroup{}

	go func(done *sync.WaitGroup) {
		defer done.Done()

		_, errCopyToParty1 := io.Copy(party1, party2)

		party1.Close()

		if errCopyToParty1 != nil {
			log.Printf(
				"handleConnection: %s -> %s error: %s",
				party2Name,
				party1Name,
				errCopyToParty1.Error())
		}
	}(waiterReference(allIoFinished))

	go func(done *sync.WaitGroup) {
		defer done.Done()

		_, errCopyToParty2 := io.Copy(party2, party1)

		party2.Close()

		if errCopyToParty2 != nil {
			log.Printf(
				"handleConnection: %s -> %s error: %s",
				party1Name,
				party2Name,
				errCopyToParty2.Error())
		}
	}(waiterReference(allIoFinished))

	allIoFinished.Wait()
}

func getCaCert() *x509.CertPool {
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM([]byte(caCert))
	return caCertPool
}

func waiterReference(wg *sync.WaitGroup) *sync.WaitGroup {
	wg.Add(1)
	return wg
}
