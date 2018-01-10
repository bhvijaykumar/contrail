package db

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/Juniper/contrail/pkg/common"
	"github.com/Juniper/contrail/pkg/generated/models"
)

func TestInterfaceRouteTable(t *testing.T) {
	t.Parallel()
	db := testDB
	common.UseTable(db, "interface_route_table")
	defer func() {
		common.ClearTable(db, "interface_route_table")
		if p := recover(); p != nil {
			panic(p)
		}
	}()
	model := models.MakeInterfaceRouteTable()
	model.UUID = "interface_route_table_dummy_uuid"
	model.FQName = []string{"default", "default-domain", "interface_route_table_dummy"}

	err := common.DoInTransaction(db, func(tx *sql.Tx) error {
		return CreateInterfaceRouteTable(tx, model)
	})
	if err != nil {
		t.Fatal("create failed", err)
	}

	err = common.DoInTransaction(db, func(tx *sql.Tx) error {
		models, err := ListInterfaceRouteTable(tx, &common.ListSpec{Limit: 1})
		if err != nil {
			return err
		}
		if len(models) != 1 {
			return fmt.Errorf("expected one element")
		}
		return nil
	})
	if err != nil {
		t.Fatal("list failed", err)
	}

	err = common.DoInTransaction(db, func(tx *sql.Tx) error {
		return DeleteInterfaceRouteTable(tx, model.UUID, nil)
	})
	if err != nil {
		t.Fatal("delete failed", err)
	}

	err = common.DoInTransaction(db, func(tx *sql.Tx) error {
		models, err := ListInterfaceRouteTable(tx, &common.ListSpec{Limit: 1})
		if err != nil {
			return err
		}
		if len(models) != 0 {
			return fmt.Errorf("expected no element")
		}
		return nil
	})
	if err != nil {
		t.Fatal("list failed", err)
	}
	return
}
