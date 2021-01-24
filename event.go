package main

import (
	"github.com/siddontang/go-mysql/schema"
	"github.com/tidwall/sjson"
)

type Event struct {
	Table  *schema.Table
	Action string
	Before []interface{}
	After  []interface{}
}

func (e *Event) String() string {
	return e.ToJson()
}

func (e *Event) ToJson() string {
	json := `{"before": null, "after": null}`
	json, _ = sjson.Set(json, "action", e.Action)
	json, _ = sjson.Set(json, "table", e.Table.Name)

	var before, after string

	before, _ = rowToJson(e.Table, e.Before)
	if before != "" {
		json, _ = sjson.SetRaw(json, "before", before)
	}

	after, _ = rowToJson(e.Table, e.After)
	if after != "" {
		json, _ = sjson.SetRaw(json, "after", after)
	}

	return json
}

func rowToJson(table *schema.Table, row []interface{}) (string, error) {
	json := "{}"

	for _, col := range table.Columns {
		val, err := table.GetColumnValue(col.Name, row)
		if err != nil {
			return "", err
		}

		json, _ = sjson.Set(json, col.Name, val)
	}

	return json, nil
}
