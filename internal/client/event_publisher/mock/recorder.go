package mock

type callRecorder struct {
	calls []string
}

func (r *callRecorder) record(name string) {
	r.calls = append(r.calls, name)
}
