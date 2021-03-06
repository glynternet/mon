package postgres

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/glynternet/go-accounting/account"
	"github.com/glynternet/go-money/currency"
	"github.com/glynternet/mon/pkg/storage"
	"github.com/lib/pq"
	"github.com/pkg/errors"
)

const (
	fieldID       = "id"
	fieldName     = "name"
	fieldOpened   = "opened"
	fieldClosed   = "closed"
	fieldCurrency = "currency"
	fieldDeleted  = "deleted"
	accountsTable = "accounts"
)

var (
	accountsFieldsInsert = fmt.Sprintf(
		"%s, %s, %s, %s",
		fieldName,
		fieldOpened,
		fieldClosed,
		fieldCurrency)

	accountsFieldsSelect = fmt.Sprintf(
		"%s, %s, %s, %s, %s, %s",
		fieldID,
		fieldName,
		fieldOpened,
		fieldClosed,
		fieldCurrency,
		fieldDeleted)

	accountsSelectPrefix = fmt.Sprintf(
		`SELECT %s FROM %s WHERE %s IS NULL `,
		accountsFieldsSelect,
		accountsTable,
		fieldDeleted)

	querySelectAccounts = fmt.Sprintf(
		"%sORDER BY %s ASC;",
		accountsSelectPrefix,
		fieldID)

	querySelectAccount = fmt.Sprintf(
		"%sAND %s = $1;",
		accountsSelectPrefix,
		fieldID)

	queryInsertAccount = fmt.Sprintf(
		`INSERT INTO %s (%s) VALUES ($1, $2, $3, $4) returning %s`,
		accountsTable,
		accountsFieldsInsert,
		accountsFieldsSelect)

	queryUpdateAccount = fmt.Sprintf(
		`UPDATE %s SET %s = $1, %s = $2, %s = $3, %s = $4 WHERE %s = $5 returning %s`,
		accountsTable,
		fieldName,
		fieldOpened,
		fieldClosed,
		fieldCurrency,
		fieldID,
		accountsFieldsSelect)

	queryDeleteAccount = fmt.Sprintf(
		`UPDATE %s SET %s = $1 WHERE %s = $2`,
		accountsTable,
		fieldDeleted,
		fieldID,
	)
)

// SelectAccounts returns an Accounts item holding all Account entries within
// the given database along with any errors that occurred whilst attempting to
// retrieve the Accounts.
func (pg postgres) SelectAccounts() (*storage.Accounts, error) {
	return queryAccounts(pg.db, querySelectAccounts)
}

// SelectAccount returns an Account with the given id.
func (pg postgres) SelectAccount(id uint) (*storage.Account, error) {
	dba, err := queryAccount(pg.db, querySelectAccount, id)
	return dba, errors.Wrap(err, "querying Account")
}

// InsertAccount inserts an account.Account in the storage backend and returns it.
func (pg postgres) InsertAccount(a account.Account) (*storage.Account, error) {
	dba, err := queryAccount(pg.db, queryInsertAccount, a.Name(), a.Opened(), pq.NullTime(a.Closed()), a.CurrencyCode())
	return dba, errors.Wrap(err, "querying Account")
}

// UpdateAccount updates the account at a given id with the values from the given account.Account
func (pg postgres) UpdateAccount(id uint, updates account.Account) (*storage.Account, error) {
	return queryAccount(
		pg.db,
		queryUpdateAccount,
		updates.Name(),
		updates.Opened(),
		pq.NullTime(updates.Closed()),
		updates.CurrencyCode(),
		id,
	)
}

// DeleteAccount deletes an account with the given id
func (pg postgres) DeleteAccount(id uint) error {
	r, err := pg.db.Exec(queryDeleteAccount, time.Now(), id)
	if err != nil {
		return errors.Wrap(err, "executing query")
	}
	n, err := r.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "getting number of rows affected ")
	}
	if n != 1 {
		return fmt.Errorf("expected 1 affected row but got %d", n)
	}
	return nil
}

// do not export this function.
// The use of interface{} here is deemed to be acceptable here because it is only used within the
// context of this package
func queryAccount(db *sql.DB, queryString string, values ...interface{}) (*storage.Account, error) {
	as, err := queryAccounts(db, queryString, values...)
	if err != nil {
		return nil, errors.Wrap(err, "querying accounts")
	}
	resLen := len(*as)
	if resLen == 0 {
		return nil, errors.New("query returned no accounts")
	}
	if resLen > 1 {
		return nil, fmt.Errorf("expected 1 account but query returned %d", resLen)
	}
	return &(*as)[0], nil
}

func queryAccounts(db *sql.DB, queryString string, values ...interface{}) (*storage.Accounts, error) {
	rows, err := db.Query(queryString, values...)
	if err != nil {
		return nil, err
	}
	defer nonReturningCloseRows(rows)
	return scanRowsForAccounts(rows)
}

// scanRowsForAccounts scans an sql.Rows object for storage.Accounts objects
// and returns then along with any error that occurs whilst attempting to scan.
func scanRowsForAccounts(rows *sql.Rows) (*storage.Accounts, error) {
	var openAccounts storage.Accounts
	for rows.Next() {
		var id uint
		var name, code string
		var opened time.Time
		var closed, deleted pq.NullTime
		// 	fieldID, fieldName, fieldOpened, fieldClosed, fieldCurrency, fieldDeleted)
		err := rows.Scan(&id, &name, &opened, &closed, &code, &deleted)
		if err != nil {
			return nil, errors.Wrap(err, "scanning row")
		}
		c, err := currency.NewCode(code)
		if err != nil {
			return nil, errors.Wrap(err, "generating new currency code")
		}
		innerAccount, err := account.New(name, *c, opened)
		if err != nil {
			return nil, errors.Wrap(err, "creating new inner account")
		}
		if closed.Valid {
			err = account.CloseTime(closed.Time)(innerAccount)
			if err != nil {
				return nil, errors.Wrap(err, "applying closed time to inner account")
			}
		}
		a := &storage.Account{ID: id, Account: *innerAccount}
		if deleted.Valid {
			err := storage.DeletedAt(deleted.Time)(a)
			if err != nil {
				return nil, errors.Wrap(err, "applying deleted time to inner account")
			}
		}
		openAccounts = append(openAccounts, *a)
	}
	return &openAccounts, rows.Err()
}
