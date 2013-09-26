package main

import (
	"bytes"
	"flag"
	"fmt"
	"net/mail"
	"net/smtp"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const Version = "0.1"

var flagHelp bool
var flagHost string
var flagInterval int64
var flagMaxErrors int64
var flagSMTP string
var flagSMTPFrom string
var flagSMTPTLS bool
var flagSMTPTo string
var flagVerbose bool

func sendMail() {
	if flagSMTP == "" {
		return
	}

	c, err := smtp.Dial(flagSMTP)

	if err != nil {
		v("Cannot open SMTP connection: %v\n", err)

		return
	}

	c.Mail(flagSMTPFrom)
	c.Rcpt(flagSMTPTo)

	wc, err := c.Data()

	if err != nil {
		v("Cannot open Data writer: %v\n", err)

		return
	}

	defer wc.Close()

	buf := bytes.NewBufferString(`Return-path: <` + flagSMTPFrom + `>
From: ` + flagSMTPFrom + `
To: ` + flagSMTPTo + `
Subject: ` + flagHost + ` is down
Content-Transfer-Encoding: 7Bit
Content-Type: text/plain; charset="us-ascii"

` + flagHost + ` is not reachable via ping
`)

	if _, err = buf.WriteTo(wc); err != nil {
		v("Cannot write mail body: %v\n", err)
	}

	v("Mail sent\n")
}

func v(format string, a ...interface{}) {
	if flagVerbose {
		fmt.Fprintf(os.Stderr, format, a...)
	}
}

func main() {
	flag.BoolVar(&flagHelp, "help", false, "Show this help")
	flag.StringVar(&flagHost, "host", "", "The host to ping")
	flag.Int64Var(&flagInterval, "interval", 60, "Ping interval in seconds")
	flag.Int64Var(&flagMaxErrors, "max-errors", 5, "How many pings can fail before a report is sent")
	flag.StringVar(&flagSMTP, "smtp", "", "The SMTP server + port for sending report mails")
	flag.StringVar(&flagSMTPFrom, "smtp-from", "", "From-mail address")
	flag.BoolVar(&flagSMTPTLS, "smtp-tls", false, "Use TLS for the SMTP connection")
	flag.StringVar(&flagSMTPTo, "smtp-to", "", "To-mail address")
	flag.BoolVar(&flagVerbose, "verbose", false, "Do verbose output")

	flag.Parse()

	if flagHost == "" || flagInterval <= 0 || flagMaxErrors <= 0 || flagHelp {
		fmt.Printf("glamor v%s\n", Version)
		fmt.Printf("usage:\n")
		fmt.Printf("\t%s -host <host> -interval <interval>\n", os.Args[0])
		fmt.Printf("options\n")
		flag.PrintDefaults()
		fmt.Printf("\n")

		if !flagHelp {
			panic("Wrong arguments")
		}
	}

	if _, err := mail.ParseAddress(flagSMTPFrom); flagSMTPFrom != "" && err != nil {
		panic("smtp-from is not a valid mail address")
	} else if _, err := mail.ParseAddress(flagSMTPTo); flagSMTPFrom != "" && err != nil {
		panic("smtp-to is not a valid mail address")
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	go func() {
		s := <-sig
		v("Caught signal \"%v\", will exit now\n", s)

		os.Exit(0)
	}()

	var sentMail = false
	var errors int64 = 0

	for {
		var cmd = exec.Command("ping", []string{"-c", "1", flagHost}...)

		out, err := cmd.CombinedOutput()

		if err != nil {
			v("Ping failed to %s: %v\n", flagHost, err)
		}

		if strings.Contains(string(out), "1 received") {
			errors--

			if errors < 100 {
				errors = 0
				sentMail = false

				v("Reset mail sent flag\n")
			}
		} else {
			errors++

			if errors >= flagMaxErrors {
				errors = 0

				v("Reached error count\n")

				if !sentMail {
					sendMail()

					sentMail = true
				}
			}
		}

		time.Sleep(time.Duration(flagInterval) * time.Second)
	}

	return
}
