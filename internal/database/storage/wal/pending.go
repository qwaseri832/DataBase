package wal

import "github.com/qwaseri832/DataBase/internal/syncx"

type Pending struct {
	rec     Record
	promise *syncx.Promise[error]
}

func newPending(lsn int64, op Op, args []string) Pending {
	p := syncx.NewPromise[error]()
	return Pending{
		rec:     Record{LSN: lsn, Op: op, Args: args},
		promise: p,
	}
}

func (p *Pending) Record() Record              { return p.rec }
func (p *Pending) Done(err error)              { p.promise.Resolve(err) }
func (p *Pending) Future() syncx.Future[error] { return p.promise.Future() }
