package handler

import (
	"errors"

	"github.com/josehenrique-dev/rinha-2026/internal/vectorize"
)

func ParsePayload(buf []byte, p *vectorize.Payload) error {
	*p = vectorize.Payload{}
	s := scanner{buf: buf}
	if !s.openObject() {
		return errors.New("json: expected '{'")
	}
	for {
		s.skipWS()
		if s.done() {
			return errors.New("json: unexpected EOF")
		}
		if s.peek() == '}' {
			s.advance(1)
			return nil
		}
		key, err := s.readString()
		if err != nil {
			return err
		}
		if !s.expect(':') {
			return errors.New("json: expected ':' after key")
		}
		s.skipWS()

		switch string(key) {
		case "id":
			if _, err := s.readString(); err != nil {
				return err
			}
		case "transaction":
			if err := parseTransaction(&s, &p.Transaction); err != nil {
				return err
			}
		case "customer":
			if err := parseCustomer(&s, &p.Customer); err != nil {
				return err
			}
		case "merchant":
			if err := parseMerchant(&s, &p.Merchant); err != nil {
				return err
			}
		case "terminal":
			if err := parseTerminal(&s, &p.Terminal); err != nil {
				return err
			}
		case "last_transaction":
			if s.peek() == 'n' {
				if !s.expectLiteral("null") {
					return errors.New("json: expected null")
				}
				p.LastTransaction = nil
			} else {
				lt := &vectorize.LastTransaction{}
				if err := parseLastTx(&s, lt); err != nil {
					return err
				}
				p.LastTransaction = lt
			}
		default:
			if err := s.skipValue(); err != nil {
				return err
			}
		}

		s.skipWS()
		if s.peek() == ',' {
			s.advance(1)
			continue
		}
		if s.peek() == '}' {
			s.advance(1)
			return nil
		}
		return errors.New("json: expected ',' or '}'")
	}
}

func parseTransaction(s *scanner, t *vectorize.Transaction) error {
	if !s.openObject() {
		return errors.New("json: expected '{' for transaction")
	}
	for {
		s.skipWS()
		if s.peek() == '}' {
			s.advance(1)
			return nil
		}
		key, err := s.readString()
		if err != nil {
			return err
		}
		if !s.expect(':') {
			return errors.New("json: expected ':'")
		}
		s.skipWS()
		switch string(key) {
		case "amount":
			f, err := s.readFloat()
			if err != nil {
				return err
			}
			t.Amount = f
		case "installments":
			n, err := s.readInt()
			if err != nil {
				return err
			}
			t.Installments = n
		case "requested_at":
			str, err := s.readString()
			if err != nil {
				return err
			}
			t.RequestedAt = string(str)
		default:
			if err := s.skipValue(); err != nil {
				return err
			}
		}
		s.skipWS()
		if s.peek() == ',' {
			s.advance(1)
			continue
		}
		if s.peek() == '}' {
			s.advance(1)
			return nil
		}
		return errors.New("json: expected ',' or '}'")
	}
}

func parseCustomer(s *scanner, c *vectorize.Customer) error {
	if !s.openObject() {
		return errors.New("json: expected '{' for customer")
	}
	for {
		s.skipWS()
		if s.peek() == '}' {
			s.advance(1)
			return nil
		}
		key, err := s.readString()
		if err != nil {
			return err
		}
		if !s.expect(':') {
			return errors.New("json: expected ':'")
		}
		s.skipWS()
		switch string(key) {
		case "avg_amount":
			f, err := s.readFloat()
			if err != nil {
				return err
			}
			c.AvgAmount = f
		case "tx_count_24h":
			n, err := s.readInt()
			if err != nil {
				return err
			}
			c.TxCount24h = n
		case "known_merchants":
			if err := parseStringArray(s, &c.KnownMerchants); err != nil {
				return err
			}
		default:
			if err := s.skipValue(); err != nil {
				return err
			}
		}
		s.skipWS()
		if s.peek() == ',' {
			s.advance(1)
			continue
		}
		if s.peek() == '}' {
			s.advance(1)
			return nil
		}
		return errors.New("json: expected ',' or '}'")
	}
}

func parseMerchant(s *scanner, m *vectorize.Merchant) error {
	if !s.openObject() {
		return errors.New("json: expected '{' for merchant")
	}
	for {
		s.skipWS()
		if s.peek() == '}' {
			s.advance(1)
			return nil
		}
		key, err := s.readString()
		if err != nil {
			return err
		}
		if !s.expect(':') {
			return errors.New("json: expected ':'")
		}
		s.skipWS()
		switch string(key) {
		case "id":
			str, err := s.readString()
			if err != nil {
				return err
			}
			m.ID = string(str)
		case "mcc":
			str, err := s.readString()
			if err != nil {
				return err
			}
			m.MCC = string(str)
		case "avg_amount":
			f, err := s.readFloat()
			if err != nil {
				return err
			}
			m.AvgAmount = f
		default:
			if err := s.skipValue(); err != nil {
				return err
			}
		}
		s.skipWS()
		if s.peek() == ',' {
			s.advance(1)
			continue
		}
		if s.peek() == '}' {
			s.advance(1)
			return nil
		}
		return errors.New("json: expected ',' or '}'")
	}
}

func parseTerminal(s *scanner, t *vectorize.Terminal) error {
	if !s.openObject() {
		return errors.New("json: expected '{' for terminal")
	}
	for {
		s.skipWS()
		if s.peek() == '}' {
			s.advance(1)
			return nil
		}
		key, err := s.readString()
		if err != nil {
			return err
		}
		if !s.expect(':') {
			return errors.New("json: expected ':'")
		}
		s.skipWS()
		switch string(key) {
		case "is_online":
			b, err := s.readBool()
			if err != nil {
				return err
			}
			t.IsOnline = b
		case "card_present":
			b, err := s.readBool()
			if err != nil {
				return err
			}
			t.CardPresent = b
		case "km_from_home":
			f, err := s.readFloat()
			if err != nil {
				return err
			}
			t.KmFromHome = f
		default:
			if err := s.skipValue(); err != nil {
				return err
			}
		}
		s.skipWS()
		if s.peek() == ',' {
			s.advance(1)
			continue
		}
		if s.peek() == '}' {
			s.advance(1)
			return nil
		}
		return errors.New("json: expected ',' or '}'")
	}
}

func parseLastTx(s *scanner, lt *vectorize.LastTransaction) error {
	if !s.openObject() {
		return errors.New("json: expected '{' for last_transaction")
	}
	for {
		s.skipWS()
		if s.peek() == '}' {
			s.advance(1)
			return nil
		}
		key, err := s.readString()
		if err != nil {
			return err
		}
		if !s.expect(':') {
			return errors.New("json: expected ':'")
		}
		s.skipWS()
		switch string(key) {
		case "timestamp":
			str, err := s.readString()
			if err != nil {
				return err
			}
			lt.Timestamp = string(str)
		case "km_from_current":
			f, err := s.readFloat()
			if err != nil {
				return err
			}
			lt.KmFromCurrent = f
		default:
			if err := s.skipValue(); err != nil {
				return err
			}
		}
		s.skipWS()
		if s.peek() == ',' {
			s.advance(1)
			continue
		}
		if s.peek() == '}' {
			s.advance(1)
			return nil
		}
		return errors.New("json: expected ',' or '}'")
	}
}

func parseStringArray(s *scanner, out *[]string) error {
	*out = (*out)[:0]
	s.skipWS()
	if !s.expect('[') {
		return errors.New("json: expected '['")
	}
	s.skipWS()
	if s.peek() == ']' {
		s.advance(1)
		return nil
	}
	for {
		s.skipWS()
		str, err := s.readString()
		if err != nil {
			return err
		}
		*out = append(*out, string(str))
		s.skipWS()
		if s.peek() == ',' {
			s.advance(1)
			continue
		}
		if s.peek() == ']' {
			s.advance(1)
			return nil
		}
		return errors.New("json: expected ',' or ']'")
	}
}

type scanner struct {
	buf []byte
	pos int
}

func (s *scanner) done() bool { return s.pos >= len(s.buf) }

func (s *scanner) peek() byte {
	if s.done() {
		return 0
	}
	return s.buf[s.pos]
}

func (s *scanner) advance(n int) { s.pos += n }

func (s *scanner) skipWS() {
	for !s.done() {
		switch s.buf[s.pos] {
		case ' ', '\t', '\r', '\n':
			s.pos++
		default:
			return
		}
	}
}

func (s *scanner) expect(b byte) bool {
	s.skipWS()
	if s.done() || s.buf[s.pos] != b {
		return false
	}
	s.pos++
	return true
}

func (s *scanner) expectLiteral(lit string) bool {
	s.skipWS()
	if s.pos+len(lit) > len(s.buf) {
		return false
	}
	if string(s.buf[s.pos:s.pos+len(lit)]) != lit {
		return false
	}
	s.pos += len(lit)
	return true
}

func (s *scanner) openObject() bool { return s.expect('{') }

func (s *scanner) readString() ([]byte, error) {
	s.skipWS()
	if !s.expect('"') {
		return nil, errors.New("json: expected '\"'")
	}
	start := s.pos
	for s.pos < len(s.buf) && s.buf[s.pos] != '"' {
		if s.buf[s.pos] == '\\' {
			s.pos += 2
			continue
		}
		s.pos++
	}
	if s.pos >= len(s.buf) {
		return nil, errors.New("json: unterminated string")
	}
	end := s.pos
	s.pos++
	return s.buf[start:end], nil
}

func (s *scanner) readInt() (int, error) {
	s.skipWS()
	start := s.pos
	if !s.done() && s.buf[s.pos] == '-' {
		s.pos++
	}
	for !s.done() && s.buf[s.pos] >= '0' && s.buf[s.pos] <= '9' {
		s.pos++
	}
	if s.pos == start {
		return 0, errors.New("json: expected integer")
	}
	return parseInt32Bytes(s.buf[start:s.pos])
}

func (s *scanner) readFloat() (float32, error) {
	s.skipWS()
	start := s.pos
	if !s.done() && s.buf[s.pos] == '-' {
		s.pos++
	}
	for !s.done() {
		c := s.buf[s.pos]
		if (c >= '0' && c <= '9') || c == '.' || c == 'e' || c == 'E' || c == '+' || c == '-' {
			s.pos++
			continue
		}
		break
	}
	if s.pos == start {
		return 0, errors.New("json: expected number")
	}
	return parseFloat32Bytes(s.buf[start:s.pos])
}

func (s *scanner) readBool() (bool, error) {
	s.skipWS()
	if s.expectLiteral("true") {
		return true, nil
	}
	if s.expectLiteral("false") {
		return false, nil
	}
	return false, errors.New("json: expected bool")
}

func (s *scanner) skipValue() error {
	s.skipWS()
	if s.done() {
		return errors.New("json: unexpected EOF")
	}
	switch s.peek() {
	case '"':
		_, err := s.readString()
		return err
	case '{':
		return s.skipObject()
	case '[':
		return s.skipArray()
	case 't':
		if !s.expectLiteral("true") {
			return errors.New("json: expected true")
		}
		return nil
	case 'f':
		if !s.expectLiteral("false") {
			return errors.New("json: expected false")
		}
		return nil
	case 'n':
		if !s.expectLiteral("null") {
			return errors.New("json: expected null")
		}
		return nil
	default:
		_, err := s.readFloat()
		return err
	}
}

func (s *scanner) skipObject() error {
	if !s.expect('{') {
		return errors.New("json: expected '{'")
	}
	for {
		s.skipWS()
		if s.peek() == '}' {
			s.advance(1)
			return nil
		}
		if _, err := s.readString(); err != nil {
			return err
		}
		if !s.expect(':') {
			return errors.New("json: expected ':'")
		}
		if err := s.skipValue(); err != nil {
			return err
		}
		s.skipWS()
		if s.peek() == ',' {
			s.advance(1)
			continue
		}
		if s.peek() == '}' {
			s.advance(1)
			return nil
		}
		return errors.New("json: expected ',' or '}'")
	}
}

func (s *scanner) skipArray() error {
	if !s.expect('[') {
		return errors.New("json: expected '['")
	}
	s.skipWS()
	if s.peek() == ']' {
		s.advance(1)
		return nil
	}
	for {
		if err := s.skipValue(); err != nil {
			return err
		}
		s.skipWS()
		if s.peek() == ',' {
			s.advance(1)
			continue
		}
		if s.peek() == ']' {
			s.advance(1)
			return nil
		}
		return errors.New("json: expected ',' or ']'")
	}
}

func parseInt32Bytes(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, errors.New("json: empty integer")
	}
	i := 0
	neg := false
	if b[0] == '-' {
		neg = true
		i = 1
		if len(b) == 1 {
			return 0, errors.New("json: bare '-'")
		}
	}
	var n int
	for ; i < len(b); i++ {
		c := b[i]
		if c < '0' || c > '9' {
			return 0, errors.New("json: bad digit")
		}
		n = n*10 + int(c-'0')
	}
	if neg {
		n = -n
	}
	return n, nil
}

func parseFloat32Bytes(b []byte) (float32, error) {
	if len(b) == 0 {
		return 0, errors.New("json: empty float")
	}
	i := 0
	neg := false
	if b[0] == '-' {
		neg = true
		i++
		if i == len(b) {
			return 0, errors.New("json: bare '-'")
		}
	} else if b[0] == '+' {
		i++
		if i == len(b) {
			return 0, errors.New("json: bare '+'")
		}
	}

	intStart := i
	var intVal float64
	for ; i < len(b); i++ {
		c := b[i]
		if c < '0' || c > '9' {
			break
		}
		intVal = intVal*10 + float64(c-'0')
	}
	if i == intStart {
		return 0, errors.New("json: missing integer part")
	}

	frac := 0.0
	fracDiv := 1.0
	if i < len(b) && b[i] == '.' {
		i++
		fracStart := i
		for ; i < len(b); i++ {
			c := b[i]
			if c < '0' || c > '9' {
				break
			}
			frac = frac*10 + float64(c-'0')
			fracDiv *= 10
		}
		if i == fracStart {
			return 0, errors.New("json: empty fractional part")
		}
	}

	val := intVal + frac/fracDiv

	if i < len(b) && (b[i] == 'e' || b[i] == 'E') {
		i++
		expNeg := false
		if i < len(b) && (b[i] == '-' || b[i] == '+') {
			expNeg = b[i] == '-'
			i++
		}
		expStart := i
		var expVal int
		for ; i < len(b); i++ {
			c := b[i]
			if c < '0' || c > '9' {
				break
			}
			expVal = expVal*10 + int(c-'0')
		}
		if i == expStart {
			return 0, errors.New("json: exponent missing digits")
		}
		if expNeg {
			expVal = -expVal
		}
		val *= pow10(expVal)
	}

	if i != len(b) {
		return 0, errors.New("json: trailing characters")
	}
	if neg {
		val = -val
	}
	return float32(val), nil
}

func pow10(n int) float64 {
	if n == 0 {
		return 1
	}
	v := 1.0
	if n > 0 {
		for i := 0; i < n; i++ {
			v *= 10
		}
		return v
	}
	for i := 0; i < -n; i++ {
		v /= 10
	}
	return v
}
