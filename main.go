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
	"strings"
	"syscall"
	"time"

	"github.com/jessevdk/go-flags"
)

const version = "1.0"

const (
	returnOk = iota
	returnHelp
	returnSignal
)

const (
	statusUp = iota
	statusDown
)

var opts struct {
	Host                      string `long:"host" description:"The host to ping" required:"true"`
	Interval                  uint64 `long:"interval" default:"60" description:"Ping interval in seconds"`
	MaxErrors                 uint64 `long:"max-errors" default:"5" description:"How many pings can fail before a report is sent"`
	ResetHostDown             uint64 `long:"reset-host-down" default:"20" description:"How many pings have to be successful in order to reset the host down status"`
	SMTP                      string `long:"smtp" description:"The SMTP server + port for sending report mails"`
	SMTPFrom                  string `long:"smtp-from" description:"From-mail address"`
	SMTPSkipCertificateVerify bool   `long:"smtp-skip-certificate-verify" description:"Do not verify the SMTP certificate"`
	SMTPTLS                   bool   `long:"smtp-tls" description:"Use TLS for the SMTP connection"`
	SMTPTo                    string `long:"smtp-to" description:"To-mail address"`
	Verbose                   bool   `long:"verbose" description:"Do verbose output"`
	Version                   bool   `long:"version" description:"Print the version of this program"`
}

func checkArguments() {
	p := flags.NewNamedParser("glamor", flags.HelpFlag)
	p.ShortDescription = "A daemon for monitoring hosts via ICMP echo request (ping)"
	p.AddGroup("Glamor arguments", "", &opts)

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

	if _, err := mail.ParseAddress(opts.SMTPFrom); opts.SMTPFrom != "" && err != nil {
		panic("smtp-from is not a valid mail address")
	} else if _, err := mail.ParseAddress(opts.SMTPTo); opts.SMTPFrom != "" && err != nil {
		panic("smtp-to is not a valid mail address")
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

	c.Mail(opts.SMTPFrom)
	c.Rcpt(opts.SMTPTo)

	wc, err := c.Data()
	if err != nil {
		return fmt.Errorf("cannot open Data writer: %v", err)
	}

	defer wc.Close()

	buf := bytes.NewBufferString(`Return-path: <` + opts.SMTPFrom + `>
From: ` + opts.SMTPFrom + `
To: ` + opts.SMTPTo + `
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

	var status byte
	var down uint64
	var up uint64

	for {
		var cmd = exec.Command("ping", "-w", "1", "-c", "1", opts.Host)

		out, err := cmd.CombinedOutput()
		if err != nil {
			v("ping to %s failed: %v", opts.Host, err)
		}

		if strings.Contains(string(out), "1 received") {
			if status == statusDown {
				up++

				if up >= opts.ResetHostDown {
					down = 0
					up = 0

					status = statusUp

					v("host is up")

					if opts.SMTP != "" {
						if err := sendMail(opts.Host+" is up", opts.Host+" is reachable again via ping"); err != nil {
							v("Cannot send mail: %v", err)
						}
					}
				}
			} else {
				down = 0
			}
		} else {
			down++

			if status == statusUp {
				if down >= opts.MaxErrors {
					down = 0
					up = 0

					status = statusDown

					v("host is down")

					if opts.SMTP != "" {
						if err := sendMail(opts.Host+" is down", opts.Host+" is not reachable via ping"); err != nil {
							v("Cannot send mail: %v", err)
						}
					}
				}
			} else {
				up = 0
			}
		}

		time.Sleep(time.Duration(opts.Interval) * time.Second)
	}
}
