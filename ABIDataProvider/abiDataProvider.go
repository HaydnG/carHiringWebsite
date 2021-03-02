package ABIDataProvider

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-adodb"
)

var (
	conn *sql.DB
)

func InitProvider() error {
	var err error

	conn, err = sql.Open("adodb", "Provider=Microsoft.ACE.OLEDB.12.0;Data Source=./ABIfiles/ABI_DRIVER_FRAUD.accdb")
	if err != nil {
		return err
	}
	conn.SetMaxOpenConns(8)
	conn.SetMaxIdleConns(8)
	conn.SetConnMaxLifetime(5 * time.Minute)

	return nil
}

func HasFraudulentClaim(lastName, firstNames, address string, DOB time.Time) (bool, error) {

	row := conn.QueryRow(`SELECT Count(*) FROM fraudulent_claim_data
									Where FAMILY_NAME = ? AND 
									FORENAMES = ? AND 
									DATE_OF_BIRTH = ? AND 
									ADDRESS_OF_CLAIM = ?`, lastName, firstNames, DOB.Format("02/01/2006"), address)

	rows := 0
	err := row.Scan(&rows)
	if err != nil {
		return false, err
	}

	return rows > 0, nil

	//dataList := make([]*data.InsurerColumn, 0)
	//
	//var (
	//	dob time.Time
	//	doc time.Time
	//)
	//
	//count := 0
	//for rows.Next() {
	//
	//	insurerData := &data.InsurerColumn{}
	//	dataList = append(dataList, insurerData)
	//
	//	err := rows.Scan(&insurerData.ID, &insurerData.LastName, &insurerData.FisrtName, &dob, &insurerData.Address, &doc, &insurerData.InsurerCode)
	//	if err != nil {
	//		return nil, err
	//	}
	//
	//	insurerData.DOB = *data.ConvertDate(dob)
	//	insurerData.DOC = *data.ConvertDate(doc)
	//
	//	count++
	//}
	//
	//dataList = dataList[:count]
}
