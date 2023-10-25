package dpfm_api_output_formatter

import (
	"database/sql"
	"fmt"
)

func ConvertToHeader(rows *sql.Rows) (*Header, error) {
	defer rows.Close()
	header := Header{}
	i := 0

	for rows.Next() {
		i++
		err := rows.Scan(
			&header.InspectionPlan,
			&header.IsCancelled,
		)
		if err != nil {
			fmt.Printf("err = %+v \n", err)
			return &header, err
		}

	}
	if i == 0 {
		fmt.Printf("DBに対象のレコードが存在しません。")
		return nil, nil
	}

	return &header, nil
}

func ConvertToInspection(rows *sql.Rows) (*[]Inspection, error) {
	defer rows.Close()
	inspections := make([]Inspection, 0)
	i := 0

	for rows.Next() {
		i++
		inspection := Inspection{}
		err := rows.Scan(
			&inspection.InspectionPlan,
			&inspection.Inspection,
			&inspection.IsCancelled,
		)
		if err != nil {
			fmt.Printf("err = %+v \n", err)
			return &inspections, err
		}

		inspections = append(inspections, inspection)
	}
	if i == 0 {
		fmt.Printf("DBに対象のレコードが存在しません。")
		return &inspections, nil
	}

	return &inspections, nil
}
