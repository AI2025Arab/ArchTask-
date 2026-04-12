// ArchTask Addition - DO NOT MERGE INTO VIKUNJA CORE
package migrations

import "xorm.io/xorm"

type structTaskArchExtension struct {
	ID          int64  `xorm:"pk autoincr"`
	ArchPhase   string `xorm:"varchar(20) null"`
	AIGenerated bool   `xorm:"not null default false"`
}

func (structTaskArchExtension) TableName() string {
	return "tasks"
}

func init() {
	Migrations = append(Migrations, &Migration{
		ID: "20260410001733",
		Migrate: func(x *xorm.Engine) error {
			return x.Sync2(new(structTaskArchExtension))
		},
		Rollback: func(x *xorm.Engine) error {
			// dropping columns implies manual or complex steps, ignoring for simplicity
			return nil
		},
	})
}
