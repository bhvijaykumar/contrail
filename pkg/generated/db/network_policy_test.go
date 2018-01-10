package db

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/Juniper/contrail/pkg/common"
	"github.com/Juniper/contrail/pkg/generated/models"
)

func TestNetworkPolicy(t *testing.T) {
	t.Parallel()
	db := testDB
	common.UseTable(db, "network_policy")
	defer func() {
		common.ClearTable(db, "network_policy")
		if p := recover(); p != nil {
			panic(p)
		}
	}()
	model := models.MakeNetworkPolicy()
	model.UUID = "network_policy_dummy_uuid"
	model.FQName = []string{"default", "default-domain", "network_policy_dummy"}

	err := common.DoInTransaction(db, func(tx *sql.Tx) error {
		return CreateNetworkPolicy(tx, model)
	})
	if err != nil {
		t.Fatal("create failed", err)
	}

	err = common.DoInTransaction(db, func(tx *sql.Tx) error {
		models, err := ListNetworkPolicy(tx, &common.ListSpec{Limit: 1})
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
		return DeleteNetworkPolicy(tx, model.UUID, nil)
	})
	if err != nil {
		t.Fatal("delete failed", err)
	}

	err = common.DoInTransaction(db, func(tx *sql.Tx) error {
		models, err := ListNetworkPolicy(tx, &common.ListSpec{Limit: 1})
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
