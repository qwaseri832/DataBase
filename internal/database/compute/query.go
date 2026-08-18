package compute

type Query struct {
	cmd  Command
	args []string
}

func newQuery(cmd Command, args []string) Query {
	return Query{cmd: cmd, args: args}
}

func (q Query) Cmd() Command   { return q.cmd }
func (q Query) Args() []string { return q.args }
