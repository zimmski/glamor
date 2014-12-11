// Copyright 2013-2014 The Glamor authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net/mail"
	"net/smtp"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jessevdk/go-flags"
)

const version = "1.2"

const (
	returnOk = iota
	returnHelp
	returnSignal
)

const (
	statusUp = iota
	statusDown
)

type host struct {
	name   string
	status byte
	down   uint64
	up     uint64
}

var opts struct {
	Config                    func(s string) error `long:"config" description:"INI config file" no-ini:"true"`
	ConfigWrite               string               `long:"config-write" description:"Write all arguments to an INI config file and exit" no-ini:"true"`
	Host                      []string             `long:"host" description:"A host to ping" required:"true"`
	Interval                  uint64               `long:"interval" default:"60" description:"Ping interval in seconds"`
	MaxDown                   uint64               `long:"max-down" default:"5" description:"How many pings must fail (in a row) before the host status is down"`
	MaxUp                     uint64               `long:"max-up" default:"20" description:"How many pings must succeed (in a row) before the host status is up again"`
	PingPacketSize            int                  `long:"ping-packet-size" default:"64" description:"Packet size of one ping. Minimum is 64 bytes maximum is 65535 bytes"`
	SMTP                      string               `long:"smtp" description:"The SMTP server + port for sending report mails"`
	SMTPFrom                  string               `long:"smtp-from" description:"From-mail address"`
	SMTPSkipCertificateVerify bool                 `long:"smtp-skip-certificate-verify" description:"Do not verify the SMTP certificate"`
	SMTPTLS                   bool                 `long:"smtp-tls" description:"Use TLS for the SMTP connection"`
	SMTPTo                    []string             `long:"smtp-to" description:"A To-mail address"`
	Verbose                   bool                 `long:"verbose" description:"Do verbose output"`
	Version                   bool                 `long:"version" description:"Print the version of this program" no-ini:"true"`

	configFile string
}

func checkArguments() {
	p := flags.NewNamedParser("glamor", flags.HelpFlag)
	p.ShortDescription = "A daemon for monitoring hosts via ICMP echo request (ping)"

	opts.Config = func(s string) error {
		ini := flags.NewIniParser(p)

		opts.configFile = s

		return ini.ParseFile(s)
	}

	if _, err := p.AddGroup("Glamor", "Glamor arguments", &opts); err != nil {
		panic(err)
	}

	if _, err := p.ParseArgs(os.Args); err != nil {
		if opts.Version {
			fmt.Printf("Glamor v%s\n", version)

			os.Exit(returnOk)
		}

		if e, ok := err.(*flags.Error); !ok || e.Type != flags.ErrHelp {
			panic(err)
		} else {
			p.WriteHelp(os.Stdout)

			os.Exit(returnHelp)
		}
	}

	if opts.Interval < 1 {
		panic("interval must be at least 1")
	} else if opts.MaxDown < 1 {
		panic("max-down must be at least 1")
	} else if opts.MaxUp < 1 {
		panic("max-up must be at least 1")
	}

	if opts.PingPacketSize < 64 || opts.PingPacketSize > 65535 {
		panic("ping-packet-size must be in range of [64,65535]")
	}

	if _, err := mail.ParseAddress(opts.SMTPFrom); opts.SMTPFrom != "" && err != nil {
		panic("smtp-from is not a valid mail address")
	}

	for _, m := range opts.SMTPTo {
		if _, err := mail.ParseAddress(m); m == "" || err != nil {
			panic(fmt.Sprintf("smtp-to \"%s\" is not a valid mail address", m))
		}
	}

	if opts.ConfigWrite != "" {
		ini := flags.NewIniParser(p)

		if err := ini.WriteFile(opts.ConfigWrite, flags.IniIncludeComments|flags.IniIncludeDefaults|flags.IniCommentDefaults); err != nil {
			panic(err)
		}

		os.Exit(returnOk)
	}
}

func checkHost(wg *sync.WaitGroup, host *host) {
	defer wg.Done()

	var cmd = exec.Command("ping", "-w", "1", "-c", "1", "-s", strconv.Itoa(opts.PingPacketSize-8), host.name)

	out, err := cmd.CombinedOutput()
	if err != nil {
		v("ping to %s failed: %v", host.name, err)
	}

	if strings.Contains(string(out), "1 received") {
		if host.status == statusDown {
			host.up++

			if host.up >= opts.MaxUp {
				host.down = 0
				host.up = 0

				host.status = statusUp

				subject := fmt.Sprintf("host %s is up", host.name)

				v(subject)

				if opts.SMTP != "" {
					if err := sendMail(subject, host.name+" is reachable again via ping"); err != nil {
						v("Cannot send mail: %v", err)
					}
				}
			}
		} else {
			host.down = 0
		}
	} else {
		host.down++

		if host.status == statusUp {
			if host.down >= opts.MaxDown {
				host.down = 0
				host.up = 0

				host.status = statusDown

				subject := fmt.Sprintf("host %s is down", host.name)

				v(subject)

				if opts.SMTP != "" {
					if err := sendMail(subject, host.name+" is not reachable via ping"); err != nil {
						v("Cannot send mail: %v", err)
					}
				}
			}
		} else {
			host.up = 0
		}
	}
}

func sendMail(subject string, message string) error {
	if opts.SMTP == "" {
		return fmt.Errorf("no SMTP server defined")
	}

	c, err := smtp.Dial(opts.SMTP)
	if err != nil {
		return fmt.Errorf("cannot open SMTP connection: %v", err)
	}

	if opts.SMTPTLS {
		if err := c.StartTLS(&tls.Config{InsecureSkipVerify: opts.SMTPSkipCertificateVerify}); err != nil {
			return fmt.Errorf("cannot start SMTP TLS: %v", err)
		}
	}

	if err := c.Mail(opts.SMTPFrom); err != nil {
		panic(err)
	}
	for _, m := range opts.SMTPTo {
		if err := c.Rcpt(m); err != nil {
			panic(err)
		}
	}

	wc, err := c.Data()
	if err != nil {
		return fmt.Errorf("cannot open Data writer: %v", err)
	}

	defer func() {
		if err := wc.Close(); err != nil {
			panic(err)
		}
	}()

	buf := bytes.NewBufferString(`Return-path: <` + opts.SMTPFrom + `>
From: ` + opts.SMTPFrom + `
To: ` + strings.Join(opts.SMTPTo, ", ") + `
Subject: ` + subject + `
Content-Transfer-Encoding: 7Bit
Content-Type: text/plain; charset="us-ascii"

` + message + `
`)

	if _, err = buf.WriteTo(wc); err != nil {
		return fmt.Errorf("cannot write mail body: %v", err)
	}

	return nil
}

func v(format string, a ...interface{}) {
	if opts.Verbose {
		fmt.Fprintf(os.Stderr, format+"\n", a...)
	}
}

func main() {
	checkArguments()

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	go func() {
		s := <-sig
		v("caught signal \"%v\", will exit now", s)

		os.Exit(returnSignal)
	}()

	hosts := make([]host, len(opts.Host))

	for i, name := range opts.Host {
		hosts[i].name = name
		hosts[i].status = statusUp
	}

	var wg sync.WaitGroup

	for {
		for i := range hosts {
			wg.Add(1)

			go checkHost(&wg, &hosts[i])
		}

		wg.Wait()

		time.Sleep(time.Duration(opts.Interval) * time.Second)
	}
}
