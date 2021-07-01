package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"github.com/function61/gokit/app/dynversion"
	"github.com/function61/gokit/io/bidipipe"
	"github.com/function61/gokit/net/netutil"
	"github.com/function61/gokit/os/osutil"
	"github.com/spf13/cobra"
	"inet.af/netaddr"
)

func main() {
	addr := "0.0.0.0:4431"

	app := &cobra.Command{
		Use:     os.Args[0],
		Short:   "Proxies Docker's socket over TLS",
		Version: dynversion.Version,
		Args:    cobra.NoArgs,
		Run: func(_ *cobra.Command, _ []string) {
			osutil.ExitIfError(func() error {
				hostAndPort, err := translateAddrOrPrefixWithPort(addr)
				if err != nil {
					return err
				}

				return listenAndServe(
					osutil.CancelOnInterruptOrTerminate(nil),
					hostAndPort)
			}())
		},
	}

	app.Flags().StringVarP(&addr, "addr", "", addr, "Use 100.64.0.0/10:4431 for CGNAT space (used also by Tailscale)")

	osutil.ExitIfError(app.Execute())
}

func listenAndServe(ctx context.Context, hostAndPort string) error {
	serverCertKey, err := osutil.GetenvRequiredFromBase64("SERVERCERT_KEY")
	if err != nil {
		return err
	}

	serverCert, err := tls.X509KeyPair([]byte(serverCert), serverCertKey)
	if err != nil {
		return err
	}

	tcpTlsListener, err := tls.Listen("tcp", hostAndPort, &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    getCaCert(),
	})
	if err != nil {
		return err
	}

	log.Printf("Listening on %s", hostAndPort)

	return netutil.CancelableServe(ctx, tcpTlsListener, func(conn net.Conn) {
		handleConnection(conn.(*tls.Conn))
	})
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

// "0.0.0.0:4331" => "0.0.0.0:4331"
// "100.64.0.0/10:4331" => "100.100.1.2:4331" (depending on host's assigned IP addresses)
func translateAddrOrPrefixWithPort(addrOrPrefixWithPort string) (string, error) {
	addrOrPrefix, port, err := net.SplitHostPort(addrOrPrefixWithPort)
	if err != nil {
		return "", err
	}

	host, err := addrFromAddrOrPrefix(addrOrPrefix)
	if err != nil {
		return "", err
	}

	return net.JoinHostPort(host, port), nil
}

// "0.0.0.0", "127.0.0.1" are returned as-is but prefixes like "100.64.0.0/10" are matched against
// local interface addresses to find a matching IP to bind to (e.g. say you want to bind to a VPN IP)
func addrFromAddrOrPrefix(addrOrPrefix string) (string, error) {
	if strings.Contains(addrOrPrefix, "/") { // looks like prefix
		ipPrefix, err := netaddr.ParseIPPrefix(addrOrPrefix)
		if err != nil {
			return "", err
		}

		addrs, err := net.InterfaceAddrs()
		if err != nil {
			return "", err
		}

		for _, addr := range addrs {
			if ipNet, is := addr.(*net.IPNet); is {
				ip, ok := netaddr.FromStdIP(ipNet.IP)
				if !ok { // shouldn't happen
					return "", fmt.Errorf("FromStdIP error: %v", ipNet.IP)
				}

				if ipPrefix.Contains(ip) {
					return ip.String(), nil
				}
			}
		}

		return "", fmt.Errorf("none of the interfaces have address for prefix %s", addrOrPrefix)
	} else {
		return addrOrPrefix, nil // is addr directly
	}
}
