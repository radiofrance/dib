package mock

import "sync"

type Writer struct {
	lock  sync.Locker
	bytes []byte
}

func NewWriter() *Writer {
	return &Writer{
		lock: new(sync.Mutex),
	}
}

func (r *Writer) Write(p []byte) (int, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.bytes = append(r.bytes, p...)

	return len(r.bytes), nil
}

func (r *Writer) GetString() string {
	r.lock.Lock()
	defer r.lock.Unlock()

	return string(r.bytes)
}
