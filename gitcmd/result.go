package gitcmd

type Result struct {
	Command  Command
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

func (r Result) StdoutString() string {
	return string(r.Stdout)
}

func (r Result) StderrString() string {
	return string(r.Stderr)
}
