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

func handleConnection(clientConn *tls.Conn) {
	defer clientConn.Close()

	// handshake would be automatically done on first Read() or Write()
	// call, but since we want access to PeerCertificates before we let
	// bidi pipe do its thing, we call it manually here.
	//
	// this has the added benefit of not uselessly dialing Docker socket
	// on cases where handshake fails
	if err := clientConn.Handshake(); err != nil {
		log.Printf("handleConnection: handshake failed: %s", err.Error())
		return
	}

	clientConnState := clientConn.ConnectionState()

	if clientConnState.HandshakeComplete && len(clientConnState.PeerCertificates) == 1 {
		cert := clientConn.ConnectionState().PeerCertificates[0]

		log.Printf(
			"handleConnection: %s connected (issuer %s)",
			cert.Subject.CommonName,
			cert.Issuer.CommonName)
	} else {
		log.Printf("handleConnection: unexpected situation; closing connection")
		return
	}

	log.Printf("handleConnection: got Client; dialing Docker sock")

	dockerSock, err := net.Dial("unix", dockerSockPath)
	if err != nil {
		log.Printf("handleConnection: Docker sock dial failed: %s", err.Error())
		return
	}
	defer dockerSock.Close()

	bidiPipe(clientConn, "Client", dockerSock, "Docker")

	log.Printf("handleConnection: closing")
}

func loadServerCertKeyFromEnv() ([]byte, error) {
	serverCertKeyBase64 := os.Getenv("SERVERCERT_KEY")
	if serverCertKeyBase64 == "" {
		return nil, errors.New("SERVERCERT_KEY not defined")
	}

	serverCertKey, err := base64.StdEncoding.DecodeString(serverCertKeyBase64)
	if err != nil {
		return nil, err
	}

	return serverCertKey, nil
}

func mainInternal() error {
	serverCertKey, err := loadServerCertKeyFromEnv()
	if err != nil {
		return err
	}

	serverCert, err := tls.X509KeyPair([]byte(serverCert), serverCertKey)
	if err != nil {
		return err
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

		go handleConnection(conn.(*tls.Conn))
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
				"bidiPipe: %s -> %s error: %s",
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
				"bidiPipe: %s -> %s error: %s",
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
