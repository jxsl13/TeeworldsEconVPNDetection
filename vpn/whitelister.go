package vpn

import (
	"errors"
	"fmt"
	"time"

	"github.com/nutsdb/nutsdb"
)

type Whitelister struct {
	nuts         *nutsdb.DB
	nutsBucket   string
	whitelistTTL uint32
}

func NewWhitelister(nuts *nutsdb.DB, bucket string, ttl time.Duration) *Whitelister {
	return &Whitelister{
		nuts:         nuts,
		nutsBucket:   bucket,
		whitelistTTL: uint32(ttl.Seconds()),
	}
}

func (wl *Whitelister) Exists(ip string) (found bool, err error) {
	if wl == nil {
		return false, nil
	}

	err = wl.nuts.View(func(tx *nutsdb.Tx) error {
		_, err := tx.Get(wl.nutsBucket, []byte(ip))
		if err != nil {
			return err
		}
		return nil
	})
	if err == nil {
		return true, nil
	}

	if errors.Is(err, nutsdb.ErrKeyNotFound) {
		return false, nil
	}

	return false, fmt.Errorf("failed to check if ip exists in whitelist: %w", err)
}

func (wl *Whitelister) Whitelist(ip string) error {
	if wl == nil {
		return nil
	}
	err := wl.nuts.Update(func(tx *nutsdb.Tx) error {
		return tx.Put(wl.nutsBucket, []byte(ip), []byte("no vpn"), wl.whitelistTTL)
	})
	if err != nil {
		return fmt.Errorf("failed to whitelist ip: %s: %w", ip, err)
	}
	return nil
}
