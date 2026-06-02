package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"time"

	"spider/internal/network"
	"spider/internal/tools"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:4200", "server address")
	timeout := flag.Duration("timeout", time.Minute, "idle timeout")
	bufStr := flag.String("buf", "4KB", "read buffer size")
	flag.Parse()

	bufSize, err := tools.ParseByteSize(*bufStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bad buffer size: %v\n", err)
		os.Exit(1)
	}

	client, err := network.DialTCP(*addr,
		network.WithClientTimeout(*timeout),
		network.WithClientBuffer(bufSize),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect failed: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("[spider] > ")
		if !scanner.Scan() {
			break
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		resp, err := client.Send([]byte(line))
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			break
		}

		fmt.Println(string(resp))
	}
}
