package main

import (
	"crypto/tls"
	"log"
	"net"
	"time"

	"github.com/function61/gokit/io/bidipipe"
	"github.com/function61/gokit/os/osutil"
)

func main() {
	osutil.ExitIfError(logic())
}

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
		log.Println("handleConnection: unexpected situation; closing connection")
		return
	}

	log.Printf("handleConnection: got Client; dialing Docker sock")

	dockerSock, err := net.Dial("unix", dockerSockPath)
	if err != nil {
		log.Printf("handleConnection: Docker sock dial failed: %s", err.Error())
		return
	}

	// by contract closes both sockets
	if err := bidipipe.Pipe(bidipipe.WithName("Client", clientConn), bidipipe.WithName("Docker", dockerSock)); err != nil {
		log.Println(err.Error())
	}

	log.Println("handleConnection: closing")
}

func logic() error {
	serverCertKey, err := osutil.GetenvRequiredFromBase64("SERVERCERT_KEY")
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
	if err != nil {
		return err
	}

	for {
		conn, err := tcpTlsListener.Accept()
		if err != nil {
			log.Printf("Accept() error: %s", err.Error())
			time.Sleep(1 * time.Second) // as not to go in a tight loop
			continue
		}

		go handleConnection(conn.(*tls.Conn))
	}
}
