package Hosts

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

type Hosts map[string]string // use net.IP?

func isSpace(b byte) bool {
	return b == ' ' || b == '\t'
}

func isComment(b byte) bool {
	return b == '#'
}

func isIP(b byte) bool {
	if b >= 'a' && b <= 'f' {
		return true
	}
	if b >= '0' && b <= '9' {
		return true
	}
	return b == '.' || b == ':'
}

func spaces(bs []byte, i int) int {
	for ; i < len(bs); i++ {
		if !isSpace(bs[i]) {
			break
		}
	}
	return i
}

func addr(bs []byte, i int) int {
	for ; i < len(bs); i++ {
		if isIP(bs[i]) {
			continue
		}
		if isSpace(bs[i]) {
			break
		}
		// return errror
	}
	return i
}

func domain(bs []byte, i int) int {
	for ; i < len(bs); i++ {
		if isSpace(bs[i]) {
			break
		}
	}
	return i
}

func parseLine(m Hosts, bs []byte) error {
	// strip comment
	for i := 0; i < len(bs); i++ {
		if isComment(bs[i]) {
			bs = bs[:i]
			break
		}
	}
	start := spaces(bs, 0)
	end := addr(bs, start)
	if end <= start {
		return nil
	}
	val := string(bs[start:end])
	for start < len(bs) {
		start = spaces(bs, end)
		end = domain(bs, start)
		if end <= start {
			return nil
		}
		key := string(bs[start:end])
		m[key] = val
	}
	return nil
}

func Parse(r io.Reader) (Hosts, error) {
	s := bufio.NewScanner(r)
	m := Hosts(make(map[string]string))
	for s.Scan() {
		parseLine(m, s.Bytes())
	}
	return m, s.Err()
}
