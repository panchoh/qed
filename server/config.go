/*
   Copyright 2018 Banco Bilbao Vizcaya Argentaria, S.A.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package server

import (
	"fmt"
	"net"
	"os"
	"os/user"
	"path/filepath"
)

type Config struct {
	// Unique name for this node. It identifies itself both in raft and
	// gossip clusters. If not set, fallback to hostname.
	NodeID string

	// TLS server bind address/port.
	HTTPAddr string

	// Raft communication bind address/port.
	RaftAddr string

	// Raft management server bind address/port. Useful to join the cluster
	// and get cluster information.
	MgmtAddr string

	// List of raft nodes, through which a cluster can be joined
	// (protocol://host:port).
	RaftJoinAddr []string

	// Path to storage directory.
	DBPath string

	// Path to Raft storage directory.
	RaftPath string

	// Gossip management server bind address/port.
	GossipAddr string

	// List of nodes, through which a gossip cluster can be joined (protocol://host:port).
	GossipJoinAddr []string

	// Path to the private key file used to sign snapshots.
	PrivateKeyPath string

	// Enables profiling endpoint.
	EnableProfiling bool

	// Enables tampering endpoint.
	EnableTampering bool

	// Enables metrics endpoint.
	EnableMetrics bool

	// Enable TLS service
	EnableTLS bool

	// TLS server cerificate
	SSLCertificate string

	// TLS server cerificate key
	SSLCertificateKey string
}

func DefaultConfig() *Config {
	hostname, _ := os.Hostname()
	currentDir := getCurrentDir()

	usr, _ := user.Current()
	homeDir := usr.HomeDir

	return &Config{
		NodeID:            hostname,
		HTTPAddr:          "127.0.0.1:8080",
		RaftAddr:          "127.0.0.1:9000",
		MgmtAddr:          "127.0.0.1:8090",
		RaftJoinAddr:      []string{},
		GossipAddr:        "127.0.0.1:9100",
		GossipJoinAddr:    []string{},
		DBPath:            currentDir + "/data",
		RaftPath:          currentDir + "/raft",
		EnableProfiling:   false,
		EnableTampering:   false,
		EnableMetrics:     true,
		EnableTLS:         true,
		SSLCertificate:    fmt.Sprintf("%s/.ssh/server.crt", homeDir),
		SSLCertificateKey: fmt.Sprintf("%s/.ssh/server.key", homeDir),
	}
}

func getCurrentDir() string {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(ex)
	return exPath
}

// AddrParts returns the parts of and address/port.
func addrParts(address string) (string, int, error) {
	_, _, err := net.SplitHostPort(address)
	if err != nil {
		return "", 0, err
	}

	addr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return "", 0, err
	}

	return addr.IP.String(), addr.Port, nil
}
