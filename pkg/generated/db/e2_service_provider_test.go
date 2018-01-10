package db

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/Juniper/contrail/pkg/common"
	"github.com/Juniper/contrail/pkg/generated/models"
)

func TestE2ServiceProvider(t *testing.T) {
	t.Parallel()
	db := testDB
	common.UseTable(db, "e2_service_provider")
	defer func() {
		common.ClearTable(db, "e2_service_provider")
		if p := recover(); p != nil {
			panic(p)
		}
	}()
	model := models.MakeE2ServiceProvider()
	model.UUID = "e2_service_provider_dummy_uuid"
	model.FQName = []string{"default", "default-domain", "e2_service_provider_dummy"}

	err := common.DoInTransaction(db, func(tx *sql.Tx) error {
		return CreateE2ServiceProvider(tx, model)
	})
	if err != nil {
		t.Fatal("create failed", err)
	}

	err = common.DoInTransaction(db, func(tx *sql.Tx) error {
		models, err := ListE2ServiceProvider(tx, &common.ListSpec{Limit: 1})
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
		return DeleteE2ServiceProvider(tx, model.UUID, nil)
	})
	if err != nil {
		t.Fatal("delete failed", err)
	}

	err = common.DoInTransaction(db, func(tx *sql.Tx) error {
		models, err := ListE2ServiceProvider(tx, &common.ListSpec{Limit: 1})
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
