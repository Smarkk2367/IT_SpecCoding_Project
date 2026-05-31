package redisx

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultRedisAddr = "localhost:6379"

var ErrNil = errors.New("redis nil")

type options struct {
	addr     string
	username string
	password string
	db       int
}

func Ping(ctx context.Context, rawURL string) error {
	conn, reader, err := connect(ctx, rawURL)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := sendCommand(conn, "PING"); err != nil {
		return err
	}
	if err := expectSimpleString(reader, "PONG"); err != nil {
		return fmt.Errorf("redis ping: %w", err)
	}

	return nil
}

func Get(ctx context.Context, rawURL string, key string) (string, error) {
	conn, reader, err := connect(ctx, rawURL)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	if err := sendCommand(conn, "GET", key); err != nil {
		return "", err
	}

	return readBulkString(reader)
}

func SetEX(ctx context.Context, rawURL string, key string, value string, ttl time.Duration) error {
	conn, reader, err := connect(ctx, rawURL)
	if err != nil {
		return err
	}
	defer conn.Close()

	seconds := int(ttl.Seconds())
	if seconds < 1 {
		seconds = 1
	}

	if err := sendCommand(conn, "SET", key, value, "EX", strconv.Itoa(seconds)); err != nil {
		return err
	}
	if err := expectOK(reader); err != nil {
		return fmt.Errorf("redis set: %w", err)
	}

	return nil
}

func Del(ctx context.Context, rawURL string, key string) error {
	conn, reader, err := connect(ctx, rawURL)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := sendCommand(conn, "DEL", key); err != nil {
		return err
	}
	_, err = readInteger(reader)
	return err
}

func XAdd(ctx context.Context, rawURL string, stream string, fields map[string]string) (string, error) {
	conn, reader, err := connect(ctx, rawURL)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	args := []string{"XADD", stream, "*"}
	for key, value := range fields {
		args = append(args, key, value)
	}

	if err := sendCommand(conn, args...); err != nil {
		return "", err
	}

	return readBulkString(reader)
}

func connect(ctx context.Context, rawURL string) (net.Conn, *bufio.Reader, error) {
	opts, err := parseURL(rawURL)
	if err != nil {
		return nil, nil, err
	}

	dialer := net.Dialer{Timeout: 500 * time.Millisecond}
	conn, err := dialer.DialContext(ctx, "tcp", opts.addr)
	if err != nil {
		return nil, nil, fmt.Errorf("redis dial: %w", err)
	}

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else {
		_ = conn.SetDeadline(time.Now().Add(500 * time.Millisecond))
	}

	reader := bufio.NewReader(conn)
	if opts.password != "" {
		args := []string{"AUTH", opts.password}
		if opts.username != "" {
			args = []string{"AUTH", opts.username, opts.password}
		}
		if err := sendCommand(conn, args...); err != nil {
			conn.Close()
			return nil, nil, err
		}
		if err := expectOK(reader); err != nil {
			conn.Close()
			return nil, nil, fmt.Errorf("redis auth: %w", err)
		}
	}

	if opts.db > 0 {
		if err := sendCommand(conn, "SELECT", strconv.Itoa(opts.db)); err != nil {
			conn.Close()
			return nil, nil, err
		}
		if err := expectOK(reader); err != nil {
			conn.Close()
			return nil, nil, fmt.Errorf("redis select: %w", err)
		}
	}

	return conn, reader, nil
}

func parseURL(rawURL string) (options, error) {
	if rawURL == "" {
		return options{addr: defaultRedisAddr}, nil
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return options{}, fmt.Errorf("parse redis url: %w", err)
	}
	if parsed.Scheme != "redis" {
		return options{}, fmt.Errorf("unsupported redis scheme %q", parsed.Scheme)
	}

	opts := options{addr: parsed.Host, db: 0}
	if opts.addr == "" {
		opts.addr = defaultRedisAddr
	}
	if !strings.Contains(opts.addr, ":") {
		opts.addr += ":6379"
	}

	if parsed.User != nil {
		opts.username = parsed.User.Username()
		opts.password, _ = parsed.User.Password()
		if opts.password == "" && opts.username != "" {
			opts.password = opts.username
			opts.username = ""
		}
	}

	if path := strings.TrimPrefix(parsed.Path, "/"); path != "" {
		db, err := strconv.Atoi(path)
		if err != nil {
			return options{}, fmt.Errorf("parse redis db: %w", err)
		}
		opts.db = db
	}

	return opts, nil
}

func sendCommand(conn net.Conn, args ...string) error {
	var builder strings.Builder
	builder.WriteString("*")
	builder.WriteString(strconv.Itoa(len(args)))
	builder.WriteString("\r\n")
	for _, arg := range args {
		builder.WriteString("$")
		builder.WriteString(strconv.Itoa(len(arg)))
		builder.WriteString("\r\n")
		builder.WriteString(arg)
		builder.WriteString("\r\n")
	}

	_, err := conn.Write([]byte(builder.String()))
	if err != nil {
		return fmt.Errorf("redis write: %w", err)
	}
	return nil
}

func expectOK(reader *bufio.Reader) error {
	return expectSimpleString(reader, "OK")
}

func expectSimpleString(reader *bufio.Reader, want string) error {
	line, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("redis read: %w", err)
	}
	line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")

	if strings.HasPrefix(line, "-") {
		return errors.New(strings.TrimPrefix(line, "-"))
	}
	if line != "+"+want {
		return fmt.Errorf("unexpected response %q", line)
	}
	return nil
}

func readBulkString(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("redis read: %w", err)
	}
	line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")

	if strings.HasPrefix(line, "-") {
		return "", errors.New(strings.TrimPrefix(line, "-"))
	}
	if line == "$-1" {
		return "", ErrNil
	}
	if !strings.HasPrefix(line, "$") {
		return "", fmt.Errorf("unexpected response %q", line)
	}

	length, err := strconv.Atoi(strings.TrimPrefix(line, "$"))
	if err != nil {
		return "", fmt.Errorf("parse bulk length: %w", err)
	}

	buf := make([]byte, length+2)
	if _, err := io.ReadFull(reader, buf); err != nil {
		return "", fmt.Errorf("redis read bulk: %w", err)
	}
	return string(buf[:length]), nil
}

func readInteger(reader *bufio.Reader) (int64, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return 0, fmt.Errorf("redis read: %w", err)
	}
	line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
	if strings.HasPrefix(line, "-") {
		return 0, errors.New(strings.TrimPrefix(line, "-"))
	}
	if !strings.HasPrefix(line, ":") {
		return 0, fmt.Errorf("unexpected response %q", line)
	}
	value, err := strconv.ParseInt(strings.TrimPrefix(line, ":"), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse integer: %w", err)
	}
	return value, nil
}
