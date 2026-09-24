package migrations

import (
	"log/slog"

	"gorm.io/gorm"
)

// 自定义索引 DDL：GORM AutoMigrate 无法表达部分唯一索引（partial unique index），
// 因此在 AutoMigrate 之后统一执行。PostgreSQL 与 SQLite 均支持以下语法。

// plotTransfersPendingIndex 同一地块同时只允许一条 pending 转交申请。
const plotTransfersPendingIndex = `
CREATE UNIQUE INDEX IF NOT EXISTS idx_plot_transfers_pending
ON plot_transfers(plot_id) WHERE status = 'pending'
`

// customIndexes 运行时按顺序执行的增量索引（全部幂等）。
var customIndexes = []string{
	plotTransfersPendingIndex,
}

// ApplyCustomIndexes 在 AutoMigrate 之后补齐自定义索引。
func ApplyCustomIndexes(db *gorm.DB, logger *slog.Logger) error {
	for _, ddl := range customIndexes {
		if err := db.Exec(ddl).Error; err != nil {
			return err
		}
	}
	if logger != nil {
		logger.Info("database custom indexes applied", "indexes", len(customIndexes))
	}
	return nil
}
