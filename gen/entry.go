package gen

type RawEntry struct {
	Source, Target string
}

func MakeRawEntry(s, t string) RawEntry {
	return RawEntry{s, t}
}

func EmptyRawEntry() RawEntry {
	return RawEntry{}
}

func (r RawEntry) IsEmpty() bool {
	return r.Source == "" || r.Target == ""
}
