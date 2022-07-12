// Copyright 2022 Fastly, Inc.

package geo

import (
	"bytes"
	"fmt"
	"strconv"
)

// Sentinel to mark EOF.
const nul = 0

type token byte

// The result of scan is one of these tokens.
const (
	tokenError token = iota
	tokenEOF
	tokenArrayEnd
	tokenArrayStart
	tokenBool
	tokenNumber
	tokenNull
	tokenObjectEnd
	tokenObjectStart
	tokenString
)

var tokensString = map[token]string{
	tokenError:       "Error",
	tokenEOF:         "EOF",
	tokenArrayStart:  "ArrayStart",
	tokenArrayEnd:    "ArrayEnd",
	tokenBool:        "Bool",
	tokenNumber:      "Number",
	tokenNull:        "Null",
	tokenObjectStart: "ObjectStart",
	tokenObjectEnd:   "ObjectEnd",
	tokenString:      "String",
}

func (tok token) String() string {
	return tokensString[tok]
}

// Maximum nested values we will track without error.
const stackSize = 100

type stack struct {
	item [stackSize]token
	top  int
}

func (s *stack) push(tok token) token {
	if s.top < stackSize {
		s.item[s.top] = tok
		s.top++
		return tok
	}
	return tokenError
}

func (s *stack) pop() token {
	if s.top > 0 {
		s.top--
		return s.item[s.top]
	}
	return tokenEOF
}

func (s *stack) size() int {
	return s.top
}

type scanner struct {
	srcBuf []byte
	srcPos int
	srcEnd int

	tokPos int
	tokEnd int

	token token

	stack stack
}

func newScanner(buf []byte) *scanner {
	return &scanner{
		srcBuf: buf,
		srcPos: 0,
		srcEnd: len(buf),
		tokPos: -1,
	}
}

func isSpace(ch byte) bool { return ch == '\t' || ch == '\n' || ch == '\r' || ch == ' ' }

func (s *scanner) scan() token {
	for {
		ch := s.next()

		for isSpace(ch) {
			ch = s.next()
		}

		s.tokPos = s.srcPos - 1

		var tok token
		switch ch {
		case nul:
			tok = tokenEOF
		case '[':
			tok = s.stack.push(tokenArrayStart)
		case ']':
			if s.stack.pop() == tokenArrayStart {
				tok = tokenArrayEnd
			}
		case '{':
			tok = s.stack.push(tokenObjectStart)
		case '}':
			if s.stack.pop() == tokenObjectStart {
				tok = tokenObjectEnd
			}
		case '"':
			tok = s.scanString()
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '-':
			tok = s.scanNumber()
		case ',', ':':
			continue
		default:
			var i int
			switch ch {
			case 'f':
				i = len("alse")
				tok = tokenBool
			case 'n':
				i = len("ull")
				tok = tokenNull
			case 't':
				i = len("rue")
				tok = tokenBool
			}
			if s.srcPos+i > s.srcEnd {
				tok = tokenError
				s.tokPos = -1
			} else {
				s.srcPos += i
			}
		}

		s.tokEnd = s.srcPos
		s.token = tok

		return tok
	}
}

func (s *scanner) scanNumber() token {
	for {
		ch := s.next()
		switch ch {
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
			'.', 'e', 'E', '+', '-':
		default:
			if ch != nul {
				s.srcPos--
			}
			return tokenNumber
		}
	}
}

func (s *scanner) scanString() token {
	ch := s.next()
	for ch != '"' {
		if ch == '\\' {
			ch = s.next()
			switch ch {
			case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
			case 'u':
			default:
				s.tokPos = -1
				return tokenError
			}
		} else {
			ch = s.next()
		}
	}
	return tokenString
}

func (s *scanner) skipValue() {
	if s.token != tokenArrayStart && s.token != tokenObjectStart {
		return
	}
	top := s.stack.size() - 1
	for {
		if s.stack.size() <= top {
			break
		}
		if s.scan() <= tokenEOF {
			break
		}
	}
}

func (s *scanner) next() byte {
	if s.srcPos >= s.srcEnd {
		return nul
	}
	ch := s.srcBuf[s.srcPos]
	s.srcPos++
	return ch
}

func (s *scanner) tokenString() string {
	if s.tokPos < 0 {
		return ""
	}
	return string(s.srcBuf[s.tokPos:s.tokEnd])
}

func (s *scanner) decodeInt() (int, error) {
	if s.token != tokenNumber {
		return 0, fmt.Errorf("unexpected JSON type %s", s.token)
	}
	return strconv.Atoi(s.tokenString())
}

func (s *scanner) decodeFloat() (float64, error) {
	if s.token != tokenNumber {
		return 0, fmt.Errorf("unexpected JSON type %s", s.token)
	}
	return strconv.ParseFloat(s.tokenString(), 64)
}

func (s *scanner) decodeString() (string, error) {
	if s.token != tokenString {
		return "", fmt.Errorf("unexpected JSON type %s", s.token)
	}
	buf := s.srcBuf[s.tokPos+1 : s.tokEnd-1]
	if bytes.IndexByte(buf, '\\') == -1 {
		return string(buf), nil
	}
	// TODO: handle unicode and escape codes
	return string(buf), nil
}
