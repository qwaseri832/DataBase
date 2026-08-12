package compute

import (
	"errors"
	"strings"
)

var (
	ErrEmptyQuery  = errors.New("empty query")
	ErrBadCommand  = errors.New("unknown command")
	ErrBadArgCount = errors.New("wrong number of arguments")
)

type Parser struct{}

func NewParser() *Parser { return &Parser{} }

func (p *Parser) Parse(raw string) (Query, error) {
	tokens := strings.Fields(raw)
	if len(tokens) == 0 {
		return Query{}, ErrEmptyQuery
	}

	cmd := lookupCommand(strings.ToUpper(tokens[0]))
	if cmd == CmdUnknown {
		return Query{}, ErrBadCommand
	}

	args := tokens[1:]
	if len(args) != expectedArgs(cmd) {
		return Query{}, ErrBadArgCount
	}

	return newQuery(cmd, args), nil
}
