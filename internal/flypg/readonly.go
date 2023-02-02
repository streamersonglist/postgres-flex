package flypg

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/fly-apps/postgres-flex/internal/flypg/admin"
	"github.com/jackc/pgx/v5"
)

const (
	readOnlyLockFile = "/data/readonly.lock"
	readOnlyEnabled  = "on"
	readOnlyDisabled = "off"
)

func SetReadOnly(ctx context.Context, n *Node, conn *pgx.Conn) error {
	if err := writeReadOnlyLock(); err != nil {
		return fmt.Errorf("failed to set readonly lock: %s", err)
	}

	databases, err := admin.ListDatabases(ctx, conn)
	if err != nil {
		return err
	}

	for _, db := range databases {
		// exclude administrative dbs
		if db.Name == "repmgr" || db.Name == "postgres" {
			continue
		}

		// Route configuration change through PGBouncer
		dbConn, err := n.NewPrimaryConnection(ctx, db.Name)
		if err != err {
			return fmt.Errorf("failed to establish connection to db %s: %s", db.Name, err)
		}
		defer dbConn.Close(ctx)

		// Set readonly
		if _, err = dbConn.Exec(ctx, "SET default_transaction_read_only=true;"); err != nil {
			return fmt.Errorf("failed to set readonly on db %s: %s", db.Name, err)
		}

		// Query configuration value and confirm the value change.
		var status string
		dbConn.QueryRow(ctx, "SHOW default_transaction_read_only;").Scan(&status)
		if err != nil {
			return fmt.Errorf("failed to verify readonly was unset: %s", err)
		}

		if status == readOnlyDisabled {
			return fmt.Errorf("failed to turn database '%s' readonly", db.Name)
		}
	}

	return nil
}

func UnsetReadOnly(ctx context.Context, n *Node, conn *pgx.Conn) error {
	// Skip if there's no readonly lock present
	if !ReadOnlyLockExists() {
		return nil
	}

	databases, err := admin.ListDatabases(ctx, conn)
	if err != nil {
		return err
	}

	for _, db := range databases {
		// exclude administrative dbs
		if db.Name == "repmgr" || db.Name == "postgres" {
			continue
		}

		// Route configuration change through PGBouncer
		dbConn, err := n.NewPrimaryConnection(ctx, db.Name)
		if err != err {
			return fmt.Errorf("failed to establish connection to db %s: %s", db.Name, err)
		}
		defer dbConn.Close(ctx)

		// Disable readonly
		_, err = dbConn.Exec(ctx, "SET default_transaction_read_only=false;")
		if err != nil {
			return fmt.Errorf("failed to unset readonly on db %s: %s", db.Name, err)
		}

		// Query configuration value and confirm the value change.
		var status string
		dbConn.QueryRow(ctx, "SHOW default_transaction_read_only;").Scan(&status)
		if err != nil {
			return fmt.Errorf("failed to verify readonly was unset: %s", err)
		}

		if status == readOnlyEnabled {
			return fmt.Errorf("failed to turn database '%s' read/write : %s", db.Name, err)
		}
	}

	if err := removeReadOnlyLock(); err != nil {
		return fmt.Errorf("failed to remove readonly lock: %s", err)
	}

	return nil
}

func ReadOnlyLockExists() bool {
	_, err := os.Stat(readOnlyLockFile)
	if os.IsNotExist(err) {
		return false
	}

	return true
}

func writeReadOnlyLock() error {
	if ReadOnlyLockExists() {
		return nil
	}

	if err := os.WriteFile(readOnlyLockFile, []byte(time.Now().String()), 0644); err != nil {
		return err
	}

	return nil
}

func removeReadOnlyLock() error {
	if !ReadOnlyLockExists() {
		return nil
	}

	if err := os.Remove(readOnlyLockFile); err != nil {
		return err
	}

	return nil
}