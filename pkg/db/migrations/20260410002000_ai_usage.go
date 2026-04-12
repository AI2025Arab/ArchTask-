// ArchTask Addition - DO NOT MERGE INTO VIKUNJA CORE
// Migration to create the archtask_ai_usage table for Freemium tracking.
package migrations

import "xorm.io/xorm"

type archTaskAIUsageTable struct {
	ID            int64  `xorm:"bigint autoincr not null unique pk"`
	UserID        int64  `xorm:"bigint not null INDEX"`
	OperationType string `xorm:"varchar(50) not null"`
	TokensUsed    int    `xorm:"int null default 0"`
	Created       int64  `xorm:"created not null"`
}

func (archTaskAIUsageTable) TableName() string {
	return "archtask_ai_usage"
}

func init() {
	Migrations = append(Migrations, &Migration{
		ID: "20260410002000",
		Migrate: func(x *xorm.Engine) error {
			return x.Sync2(new(archTaskAIUsageTable))
		},
		Rollback: func(x *xorm.Engine) error {
			return x.DropTables(new(archTaskAIUsageTable))
		},
	})
}
