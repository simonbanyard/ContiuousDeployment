package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

func newResolver() *net.Resolver {
	servers := []string{"8.8.8.8:53", "8.8.4.4:53"}
	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{}
			var (
				conn net.Conn
				err  error
			)
			for _, s := range servers {
				conn, err = d.DialContext(ctx, "udp", s)
				if err == nil {
					return conn, nil
				}
			}
			return nil, err
		},
	}
}

type lookupResult struct {
	text string
	err  error
}

func lookupAddresses(ctx context.Context, r *net.Resolver, domainname string) lookupResult {
	addrs, err := r.LookupIPAddr(ctx, domainname)
	if err != nil {
		return lookupResult{err: err}
	}
	lines := []string{"# Addresses"}
	for _, addr := range addrs {
		lines = append(lines, fmt.Sprintf("%s IN A %s", domainname, addr.IP))
	}
	return lookupResult{text: strings.Join(lines, "\n")}
}

func lookupMailExchangers(ctx context.Context, r *net.Resolver, domainname string) lookupResult {
	mxs, err := r.LookupMX(ctx, domainname)
	if err != nil {
		return lookupResult{err: err}
	}
	lines := []string{"# Mail Exchangers"}
	for _, mx := range mxs {
		lines = append(lines, fmt.Sprintf("%s IN MX %d %s", domainname, mx.Pref, mx.Host))
	}
	return lookupResult{text: strings.Join(lines, "\n")}
}

func lookupNameservers(ctx context.Context, r *net.Resolver, domainname string) lookupResult {
	nss, err := r.LookupNS(ctx, domainname)
	if err != nil {
		return lookupResult{err: err}
	}
	lines := []string{"# Nameservers"}
	for _, ns := range nss {
		lines = append(lines, fmt.Sprintf("%s IN NS %s", domainname, ns.Host))
	}
	return lookupResult{text: strings.Join(lines, "\n")}
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s DOMAINNAME\n", os.Args[0])
		os.Exit(1)
	}
	domainname := os.Args[1]

	r := newResolver()
	ctx := context.Background()

	results := make([]lookupResult, 3)
	var wg sync.WaitGroup
	wg.Add(3)

	go func() { defer wg.Done(); results[0] = lookupAddresses(ctx, r, domainname) }()
	go func() { defer wg.Done(); results[1] = lookupMailExchangers(ctx, r, domainname) }()
	go func() { defer wg.Done(); results[2] = lookupNameservers(ctx, r, domainname) }()

	wg.Wait()

	for _, res := range results {
		if res.err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: domain name not found %q\n", domainname)
			os.Exit(1)
		}
	}

	texts := make([]string, len(results))
	for i, res := range results {
		texts[i] = res.text
	}
	fmt.Printf("# Domain Summary for %q\n", domainname)
	fmt.Println(strings.Join(texts, "\n\n"))
}
