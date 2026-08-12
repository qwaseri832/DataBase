package compute

type Query struct {
	cmd  int
	args []string
}

func newQuery(cmd int, args []string) Query {
	return Query{cmd: cmd, args: args}
}

func (q Query) Cmd() int       { return q.cmd }
func (q Query) Args() []string { return q.args }
