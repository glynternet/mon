package model

import (
	"fmt"

	"github.com/glynternet/go-accounting/account"
	"github.com/glynternet/mon/pkg/storage"
	"github.com/pkg/errors"
)

// UpdateAccount updates a stored account to reflect the details of some other
// account data. The updates will be verified to ensure that any data to be
// used will be logically sound with the balances and other account details.
func UpdateAccount(s storage.Storage, a storage.Account, updates account.Account) (*storage.Account, error) {
	bs, err := s.SelectAccountBalances(a)
	if err != nil {
		return nil, errors.Wrap(err, "selecting Account Balances for update validation")
	}
	if bs != nil {
		for _, b := range *bs {
			err := updates.ValidateBalance(b.Balance)
			if err != nil {
				return nil, fmt.Errorf("update would make balance invalid: %v", err)
			}
		}
	}
	dba, err := s.UpdateAccount(a.ID, updates)
	return dba, errors.Wrap(err, "updating account")
}
