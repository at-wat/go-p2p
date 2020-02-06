package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/d/stdisco"
	"github.com/brendoncarroll/go-p2p/s/aggswarm"
	"github.com/brendoncarroll/go-p2p/s/natswarm"
	"github.com/brendoncarroll/go-p2p/s/sshswarm"
	"github.com/spf13/cobra"
	"github.com/syncthing/syncthing/lib/upnp"
)

var log = p2p.Logger

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func init() {
	rootCmd.AddCommand(upnpCmd)
	rootCmd.AddCommand(testConnectCmd)
}

var rootCmd = &cobra.Command{
	Use:   "p2putil",
	Short: "P2P testing and diagnostics",
}

var upnpCmd = &cobra.Command{
	Use:   "upnp-list",
	Short: "UPnP",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		timeout := 3 * time.Second

		log.Println("discovering nat devices...")
		natDevs := upnp.Discover(ctx, timeout/2, timeout)
		for _, natDev := range natDevs {
			localIP, err := natDev.GetExternalIPAddress()
			if err != nil {
				log.Error(err)
			}
			externalIP, err := natDev.GetExternalIPAddress()
			if err != nil {
				log.Error(err)
			}
			cmd.Println(localIP, externalIP, natDev.ID())
		}

		return nil
	},
}

var testConnectCmd = &cobra.Command{
	Use:   "test-connect",
	Short: "Tool for testing connectivity",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, privKey, _ := ed25519.GenerateKey(rand.Reader)

		s1, err := sshswarm.New("0.0.0.0:", privKey)
		if err != nil {
			log.Error(err)
		}

		s2 := natswarm.New(s1).(p2p.SecureAskSwarm)
		s3 := aggswarm.New(privKey, map[string]aggswarm.Transport{
			"ssh": s2,
		})

		s3.OnTell(func(m *p2p.Message) {
			ctx := context.TODO()
			s2.Tell(ctx, m.Src, m.Payload)
			log.Println("MSG:", m.Src, "->", m.Dst, " ", m.Payload)
		})

		disco := stdisco.NewClient(privKey, stdisco.PublicEndpoint)
		beac := p2p.NewBeacon(disco, s3, 5*time.Second)
		defer beac.Close()

		addrs := map[string]p2p.Addr{}
		for {
			for _, addr := range s3.LocalAddrs() {
				if _, exists := addrs[addr.Key()]; !exists {
					fmt.Println("ADDR:", addr)
					addrs[addr.Key()] = addr
				}
			}
			time.Sleep(time.Second)
		}
	},
}