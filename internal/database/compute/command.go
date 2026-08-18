package compute

type Command int

const (
	CmdUnknown Command = iota
	CmdSet
	CmdGet
	CmdDel
)

var nameToCmd = map[string]Command{
	"SET": CmdSet,
	"GET": CmdGet,
	"DEL": CmdDel,
}

var cmdArgCount = map[Command]int{
	CmdSet: 2,
	CmdGet: 1,
	CmdDel: 1,
}

func lookupCommand(name string) Command {
	if id, ok := nameToCmd[name]; ok {
		return id
	}
	return CmdUnknown
}

func expectedArgs(cmd Command) int {
	return cmdArgCount[cmd]
}
