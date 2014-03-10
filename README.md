# Glamor

Glamor ([Sindarin](https://en.wikipedia.org/wiki/Sindarin) for "echo") is a daemon for monitoring hosts via ICMP echo request (ping)

## Why?

I needed a tool at work to monitor one host via ICMP echo request and send a notification if the host is down. I did not find a tool, which I liked, right away and hence - as I am learning Go - I just wrote a tool myself. Glamor is intended to monitor our Icinga host therefore I could not simply use Nagios/Icinga/... itself.

## How does Glamor work?

There is (currently) no ICMP implementation in Go, so the system's <code>ping</code> command is used instead. If a threshold of packet loss is reached, a simple mail notification is sent via SMTP. The notification is sent only once per host down status, which will reset after a given amount of successful ICMP echo replies.

## How do I install and use Glamor?

As Glamor is written in Go you have to install Go first. Your distribution will most definitely have some packages or you can be brave and just install it yourself. Have a look at [the official documentation](http://golang.org/doc/install). Good luck!

To fetch and install Glamor just use the following command:

```bash
go get github.com/zimmski/glamor
```

After that you can compile Glamor into the binary <code>$GOBIN/glamor</code>:

```bash
go install github.com/zimmski/glamor
```

The following CLI arguments can be used:

```
      --config=                       INI config file
      --host=                         A host to ping
      --interval=                     Ping interval in seconds (60)
      --max-down=                     How many pings must fail (in a row) before the host status is down (5)
      --max-up=                       How many pings must succeed (in a row) before the host status is up again (20)
      --ping-packet-size=             Packet size of one ping. Minimum is 64 bytes maximum is 65535 bytes (64)
      --smtp=                         The SMTP server + port for sending report mails
      --smtp-from=                    From-mail address
      --smtp-skip-certificate-verify  Do not verify the SMTP certificate
      --smtp-tls                      Use TLS for the SMTP connection
      --smtp-to=                      A To-mail address
      --verbose                       Do verbose output
      --version                       Print the version of this program

  -h, --help                          Show this help message
```

Only the <code>--host</code> argument is required.

Some example arguments for Glamor:

* Monitor github.com with verbose output
  <pre><code>glamor --host github.com --verbose</code></pre>

* Monitor github.com and google.com
  <pre><code>glamor --host github.com --host google.com</code></pre>

* Monitor github.com every second with verbose output
  <pre><code>glamor --host github.com --verbose --interval 1</code></pre>

* Monitor github.com and send a mail if it is down
  <pre><code>glamor --host github.com --smtp localhost:25 --smtp-from monitoring@fake.domain --smtp-to guard@fake.domain</code></pre>

* Monitor github.com and send a mail via TLS connection but ignore “invalid” certificates
  <pre><code>glamor --host github.com --smtp localhost:25 --smtp-from monitoring@fake.domain --smtp-to guard@fake.domain --smtp-tls --smtp-skip-certificate-verify</code></pre>

## INI config file

All Glamor arguments can be definied with an INI config file specified through the <code>--config</code> CLI argument. An INI config file could for example look like this.

```ini
Host = github.com
Interval = 60
MaxDown = 5
MaxUp = 20
PingPacketSize = 64

SMTP = localhost:25
SMTPFrom = monitoring@fake.domain
SMTPTo = guard@fake.domain
SMTPTLS = true
SMTPSkipCertificateVerify = true

Verbose = true
```

Arguments defined in an INI config file are overwritten by CLI arguments.

## Can I make some feature requests?

Always! As long as they are plain and simple and not degenerate into a full blown monitoring tool.
