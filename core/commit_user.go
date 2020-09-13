package core

import (
	"fmt"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"sync"
)

type Users struct {
	signatures map[string]object.Signature
	lock       sync.Locker
}

func NewUsers(locker sync.Locker) *Users {
	if locker == nil {
		locker = &sync.Mutex{}
	}

	return &Users{
		signatures: make(map[string]object.Signature),
		lock:       locker,
	}
}

func (u *Users) Add(sig object.Signature) {
	u.lock.Lock()
	defer u.lock.Unlock()

	id := signatureID(sig)
	if !u.exists(id) {
		u.signatures[id] = sig
	}
}

func (u *Users) UniqueSignatures() []object.Signature {
	u.lock.Lock()
	defer u.lock.Unlock()

	sigs := make([]object.Signature, len(u.signatures))
	for _, signature := range u.signatures {
		sigs = append(sigs, signature)
	}
	return sigs
}

func (u *Users) exists(signatureID string) bool {
	_, ok := u.signatures[signatureID]
	return ok
}

func signatureID(sig object.Signature) string {
	return fmt.Sprintf("%s:%s", sig.Name, sig.Email)
}
