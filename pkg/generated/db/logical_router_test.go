package db

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/Juniper/contrail/pkg/common"
	"github.com/Juniper/contrail/pkg/generated/models"
)

func TestLogicalRouter(t *testing.T) {
	t.Parallel()
	db := testDB
	common.UseTable(db, "logical_router")
	defer func() {
		common.ClearTable(db, "logical_router")
		if p := recover(); p != nil {
			panic(p)
		}
	}()
	model := models.MakeLogicalRouter()
	model.UUID = "logical_router_dummy_uuid"
	model.FQName = []string{"default", "default-domain", "logical_router_dummy"}

	err := common.DoInTransaction(db, func(tx *sql.Tx) error {
		return CreateLogicalRouter(tx, model)
	})
	if err != nil {
		t.Fatal("create failed", err)
	}

	err = common.DoInTransaction(db, func(tx *sql.Tx) error {
		models, err := ListLogicalRouter(tx, &common.ListSpec{Limit: 1})
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
		return DeleteLogicalRouter(tx, model.UUID, nil)
	})
	if err != nil {
		t.Fatal("delete failed", err)
	}

	err = common.DoInTransaction(db, func(tx *sql.Tx) error {
		models, err := ListLogicalRouter(tx, &common.ListSpec{Limit: 1})
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
