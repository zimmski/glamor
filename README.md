# glamor

glamor (Elvish for "echo") is a daemon for monitoring a host via ICMP echo request (ping)

## Why?

I needed a tool at work to monitor one host via ICMP echo request and send a notification if the host is down. I did not find a tool, which I liked, right away and hence - as I am learning Go - I just wrote a tool myself. Glamor is intended to monitor our Icinga host therefore I could not simply use Nagios/Icinga/... itself.

## How does glamor work?

There is (currently) no ICMP implementation in Go, so the system's <code>ping</code> command is used instead. If a threshold of packet loss is reached, a simple mail notification is sent via SMTP. The notification is sent only once per host down status, which will reset after a given amount of successful ICMP echo replies.

## How do I install and use glamor?

As glamor is written in Go you have to install Go first. Your distribution will most definitely have some packages or you can be brave and just install it yourself. Have a look at [the official documentation](http://golang.org/doc/install). Good luck!

To fetch and install glamor just use the following command:

```bash
go get github.com/zimmski/glamor
```

After that the repository of glamor is fetched into <code>$GOPATH/src</code> and compiled into an executable to <code>$GOBIN/glamor</code>.

The following CLI arguments can be used:

```bash
  -help=false: Show this help
  -host="": The host to ping
  -interval=60: Ping interval in seconds
  -max-errors=5: How many pings can fail before a report is sent
  -reset-host-down=50: How many pings have to be successful in order to reset the host down status
  -smtp="": The SMTP server + port for sending report mails
  -smtp-from="": From-mail address
  -smtp-skip-certificate-verify=false: Do not verify the SMTP certificate
  -smtp-tls=false: Use TLS for the SMTP connection
  -smtp-to="": To-mail address
  -verbose=false: Do verbose output
```

Only the <code>-host</code> argument is required.

Some example arguments for glamor:

* Monitor github.com with verbose output
  <pre><code>glamor -host github.com -verbose</code></pre>

* Monitor github.com every second with verbose output
  <pre><code>glamor -host github.com -verbose -interval 1</code></pre>

* Monitor github.com and send a mail if it is down
  <pre><code>glamor -host google.com -smtp localhost:25 -smtp-from monitoring@fake.domain -smtp-to guard@fake.domain</code></pre>

* Monitor github.com and send a mail via TLS connection but ignore “invalid” certificates
  <pre><code>glamor -host google.com -smtp localhost:25 -smtp-from monitoring@fake.domain -smtp-to guard@fake.domain -smtp-tls -smtp-skip-certificate-verify</code></pre>

## Can I make some feature requests?

Always! As long as they are plain and simple and not degenerate into a full blown monitoring tool.

