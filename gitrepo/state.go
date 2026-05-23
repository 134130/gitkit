package gitrepo

type State struct {
	Dirty         bool
	Rebasing      bool
	ApplyingPatch bool
	CherryPicking bool
	Merging       bool
}

func (s State) Busy() bool {
	return s.Rebasing || s.ApplyingPatch || s.CherryPicking || s.Merging
}
