package db

import (
	"database/sql"
	"github.com/OpenBazaar/openbazaar-go/pb"
	"github.com/golang/protobuf/jsonpb"
	"strings"
	"sync"
)

type SalesDB struct {
	db   *sql.DB
	lock *sync.Mutex
}

func (s *SalesDB) Put(orderID string, contract pb.RicardianContract, state pb.OrderState, read bool) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	readInt := 0
	if read {
		readInt = 1
	}
	m := jsonpb.Marshaler{
		EnumsAsInts:  false,
		EmitDefaults: true,
		Indent:       "    ",
		OrigName:     false,
	}
	out, err := m.MarshalToString(&contract)

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare("insert or replace into sales(orderID, contract, state, read, date, total, thumbnail, buyerID, buyerBlockchainID, title, shippingName, shippingAddress) values(?,?,?,?,?,?,?,?,?,?,?,?)")
	if err != nil {
		return err
	}

	blockchainID := contract.BuyerOrder.BuyerID.BlockchainID
	shippingName := ""
	shippingAddress := ""
	if contract.BuyerOrder.Shipping != nil {
		shippingName = strings.ToLower(contract.BuyerOrder.Shipping.ShipTo)
		shippingAddress = strings.ToLower(contract.BuyerOrder.Shipping.Address)
	}
	defer stmt.Close()
	_, err = stmt.Exec(
		orderID,
		out,
		int(state),
		readInt,
		int(contract.BuyerOrder.Timestamp.Seconds),
		int(contract.BuyerOrder.Payment.Amount),
		contract.VendorListings[0].Item.Images[0].Hash,
		contract.BuyerOrder.BuyerID.Guid,
		blockchainID,
		strings.ToLower(contract.VendorListings[0].Item.Title),
		shippingName,
		shippingAddress,
	)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (s *SalesDB) MarkAsRead(orderID string) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	_, err := s.db.Exec("update sales set read=? where orderID=?", 1, orderID)
	if err != nil {
		return err
	}
	return nil
}

func (s *SalesDB) Delete(orderID string) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	_, err := s.db.Exec("delete from sales where orderID=?", orderID)
	if err != nil {
		return err
	}
	return nil
}

func (s *SalesDB) GetAll() ([]string, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	stm := "select orderID from sales"
	rows, err := s.db.Query(stm)
	defer rows.Close()
	if err != nil {
		return nil, err
	}
	var ret []string
	for rows.Next() {
		var orderID string
		if err := rows.Scan(&orderID); err != nil {
			return ret, err
		}
		ret = append(ret, orderID)
	}
	return ret, nil
}
