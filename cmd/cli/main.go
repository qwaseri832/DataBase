package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/qwaseri832/DataBase/internal/network"
	"github.com/qwaseri832/DataBase/internal/tools"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:4200", "server address")
	timeout := flag.Duration("timeout", time.Minute, "request timeout")
	bufStr := flag.String("buf", "4KB", "i/o buffer size")
	flag.Parse()

	if err := run(*addr, *timeout, *bufStr, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "spider-cli: %v\n", err)
		os.Exit(1)
	}
}

func run(addr string, timeout time.Duration, bufStr string, stdin io.Reader, stdout io.Writer) error {
	bufSize, err := tools.ParseByteSize(bufStr)
	if err != nil {
		return fmt.Errorf("buffer size: %w", err)
	}

	client, err := network.DialTCP(addr,
		network.WithClientTimeout(timeout),
		network.WithClientBuffer(bufSize),
	)
	if err != nil {
		return err
	}
	defer client.Close()

	scanner := bufio.NewScanner(stdin)
	for {
		fmt.Fprint(stdout, "[spider] > ")
		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		switch line {
		case "":
			continue
		case "exit", "quit":
			return nil
		}

		resp, err := client.Send([]byte(line))
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return errors.New("connection closed by server")
			}
			return err
		}
		fmt.Fprintln(stdout, string(resp))
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read input: %w", err)
	}
	fmt.Fprintln(stdout)
	return nil
}
