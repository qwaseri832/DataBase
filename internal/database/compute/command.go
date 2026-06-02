package compute

const (
	CmdUnknown = iota
	CmdSet
	CmdGet
	CmdDel
)

var nameToCmd = map[string]int{
	"SET": CmdSet,
	"GET": CmdGet,
	"DEL": CmdDel,
}

var cmdArgCount = map[int]int{
	CmdSet: 2,
	CmdGet: 1,
	CmdDel: 1,
}

func lookupCommand(name string) int {
	if id, ok := nameToCmd[name]; ok {
		return id
	}
	return CmdUnknown
}

func expectedArgs(cmd int) int {
	return cmdArgCount[cmd]
}
