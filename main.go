// Copyright 2013 The glamor authors.
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
}

func checkArguments() {
	p := flags.NewNamedParser("glamor", flags.HelpFlag)
	p.ShortDescription = "A daemon for monitoring hosts via ICMP echo request (ping)"
	p.AddGroup("Glamor arguments", "", &opts)

	if _, err := p.ParseArgs(os.Args); err != nil {
		if e, ok := err.(*flags.Error); !ok || e.Type != flags.ErrHelp {
			panic(err)
		} else {
			fmt.Printf("%+v\n", opts)
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

func sendMail() {
	if opts.SMTP == "" {
		return
	}

	c, err := smtp.Dial(opts.SMTP)
	if err != nil {
		v("Cannot open SMTP connection: %v\n", err)

		return
	}

	if opts.SMTPTLS {
		if err := c.StartTLS(&tls.Config{InsecureSkipVerify: opts.SMTPSkipCertificateVerify}); err != nil {
			v("Cannot start SMTP TLS: %v\n", err)

			return
		}
	}

	c.Mail(opts.SMTPFrom)
	c.Rcpt(opts.SMTPTo)

	wc, err := c.Data()
	if err != nil {
		v("Cannot open Data writer: %v\n", err)

		return
	}

	defer wc.Close()

	buf := bytes.NewBufferString(`Return-path: <` + opts.SMTPFrom + `>
From: ` + opts.SMTPFrom + `
To: ` + opts.SMTPTo + `
Subject: ` + opts.Host + ` is down
Content-Transfer-Encoding: 7Bit
Content-Type: text/plain; charset="us-ascii"

` + opts.Host + ` is not reachable via ping
`)

	if _, err = buf.WriteTo(wc); err != nil {
		v("Cannot write mail body: %v\n", err)
	}

	v("Mail sent\n")
}

func v(format string, a ...interface{}) {
	if opts.Verbose {
		fmt.Fprintf(os.Stderr, format, a...)
	}
}

func main() {
	checkArguments()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	go func() {
		s := <-sig
		v("Caught signal \"%v\", will exit now\n", s)

		os.Exit(returnOk)
	}()

	var sentMail = false
	var errors uint64

	for {
		var cmd = exec.Command("ping", []string{"-w", "1", "-c", "1", opts.Host}...)

		out, err := cmd.CombinedOutput()
		if err != nil {
			v("Ping to %s failed: %v\n", opts.Host, err)
		}

		if strings.Contains(string(out), "1 received") {
			errors--

			if sentMail && errors <= opts.ResetHostDown {
				errors = 0
				sentMail = false

				v("Reset mail sent opts.\n")
			}
		} else {
			errors++

			if errors >= opts.MaxErrors {
				errors = 0

				v("Reached error count\n")

				if !sentMail {
					sendMail()

					sentMail = true
				}
			}
		}

		time.Sleep(time.Duration(opts.Interval) * time.Second)
	}
}
