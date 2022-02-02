package mock

type tagArgs struct {
	Src  string
	Dest string
}

type Tagger struct {
	RecordedCallsArgs []tagArgs
}

func (r *Tagger) Tag(src, dest string) error {
	r.RecordedCallsArgs = append(r.RecordedCallsArgs, tagArgs{
		Src:  src,
		Dest: dest,
	})
	return nil
}
