package db

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/Juniper/contrail/pkg/common"
	"github.com/Juniper/contrail/pkg/generated/models"
)

func TestRoutingPolicy(t *testing.T) {
	t.Parallel()
	db := testDB
	common.UseTable(db, "routing_policy")
	defer func() {
		common.ClearTable(db, "routing_policy")
		if p := recover(); p != nil {
			panic(p)
		}
	}()
	model := models.MakeRoutingPolicy()
	model.UUID = "routing_policy_dummy_uuid"
	model.FQName = []string{"default", "default-domain", "routing_policy_dummy"}

	err := common.DoInTransaction(db, func(tx *sql.Tx) error {
		return CreateRoutingPolicy(tx, model)
	})
	if err != nil {
		t.Fatal("create failed", err)
	}

	err = common.DoInTransaction(db, func(tx *sql.Tx) error {
		models, err := ListRoutingPolicy(tx, &common.ListSpec{Limit: 1})
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
		return DeleteRoutingPolicy(tx, model.UUID, nil)
	})
	if err != nil {
		t.Fatal("delete failed", err)
	}

	err = common.DoInTransaction(db, func(tx *sql.Tx) error {
		models, err := ListRoutingPolicy(tx, &common.ListSpec{Limit: 1})
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
