package wal

import "spider/internal/syncx"

// Pending оборачивает Record и Promise для уведомления вызывающего.
type Pending struct {
	rec     Record
	promise *syncx.Promise[error]
}

func newPending(lsn int64, cmd int, args []string) Pending {
	p := syncx.NewPromise[error]()
	return Pending{
		rec:     Record{LSN: lsn, Cmd: cmd, Args: args},
		promise: p,
	}
}

func (p *Pending) Record() Record              { return p.rec }
func (p *Pending) Done(err error)              { p.promise.Resolve(err) }
func (p *Pending) Future() syncx.Future[error] { return p.promise.Future() }
