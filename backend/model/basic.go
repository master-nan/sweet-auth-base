package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CustomTime time.Time

func (t CustomTime) IsZero() bool {
	return time.Time(t).IsZero()
}

func (t *CustomTime) UnmarshalJSON(data []byte) (err error) {
	now, err := time.ParseInLocation(`"`+time.DateTime+`"`, string(data), time.Local)
	*t = CustomTime(now)
	return
}
func (t CustomTime) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0, len(time.DateTime)+2)
	b = append(b, '"')
	b = time.Time(t).AppendFormat(b, time.DateTime)
	b = append(b, '"')
	return b, nil
}
func (t CustomTime) String() string {
	return time.Time(t).Format(time.DateTime)
}

func (t CustomTime) Format(s string) string {
	return time.Time(t).Format(s)
}

func (t CustomTime) Value() (driver.Value, error) {
	return time.Time(t), nil
}

func (t *CustomTime) Scan(value interface{}) error {
	if value == nil {
		*t = CustomTime(time.Time{})
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		*t = CustomTime(v)
	case []byte:
		parsedTime, err := time.Parse(time.RFC3339, string(v))
		if err != nil {
			return err
		}
		*t = CustomTime(parsedTime)
	case string:
		parsedTime, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return err
		}
		*t = CustomTime(parsedTime)
	default:
		return fmt.Errorf("unsupported scan type for CustomTime: %T", value)
	}
	return nil
}

// StringSlice 自定义字符串切片类型
type StringSlice []string

// Value 实现 driver.Valuer 接口
func (s StringSlice) Value() (driver.Value, error) {
	return json.Marshal(s)
}

// Scan 实现 sql.Scanner 接口
func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = []string{}
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal StringSlice value: %v", value)
	}
	return json.Unmarshal(b, s)
}

type Basic struct {
	Id         int            `gorm:"primaryKey;type:bigint;comment:ID" json:"id"`
	GmtCreate  CustomTime     `gorm:"type:datetime;autoCreateTime;comment:创建时间" json:"gmt_create"`
	CreateUser *int           `gorm:"comment:创建人ID" json:"createUser"`
	CreateName *string        `gorm:"size:128;comment:创建人" json:"createName"`
	GmtModify  CustomTime     `gorm:"type:datetime;autoCreateTime;autoUpdateTime;comment:修改时间" json:"gmt_modify"`
	ModifyUser *int           `gorm:"column:modify_user;comment:修改人ID" json:"modify_user"`
	ModifyName *string        `gorm:"size:128;comment:修改人" json:"modify_name"`
	GmtDelete  gorm.DeletedAt `gorm:"type:datetime;comment:删除时间" json:"-"`
	DeleteUser *int           `gorm:"column:delete_user;comment:删除人ID" json:"delete_user"`
	DeleteName *string        `gorm:"size:128;comment:删除人" json:"-"`
	State      bool           `gorm:"default:true;comment:状态" json:"state"`
}

func (b *Basic) BeforeCreate(tx *gorm.DB) (err error) {
	ctx, ok := tx.Statement.Context.(*gin.Context)
	if ok {
		if userValue, exists := ctx.Get("user"); exists {
			if user, ok := userValue.(SysUser); ok {
				tx.Statement.SetColumn("create_user", user.Id)
			}
		}
	}
	return
}

func (b *Basic) BeforeUpdate(tx *gorm.DB) error {
	ctx, ok := tx.Statement.Context.(*gin.Context)
	if ok {
		if userValue, exists := ctx.Get("user"); exists {
			if user, ok := userValue.(SysUser); ok {
				tx.Statement.SetColumn("modify_user", user.Id)
			}
		}
	}
	return nil
}

func (b *Basic) BeforeDelete(tx *gorm.DB) error {
	ctx, ok := tx.Statement.Context.(*gin.Context)
	if ok {
		if userValue, exists := ctx.Get("user"); exists {
			if user, ok := userValue.(SysUser); ok {
				tx.Statement.AddClause(clause.Update{})
				tx.Statement.AddClause(clause.Set{
					{Column: clause.Column{Name: "delete_user"}, Value: user.Id},
					{Column: clause.Column{Name: "gmt_delete"}, Value: time.Now()},
				})
				tx.Statement.Build(
					clause.Update{}.Name(),
					clause.Set{}.Name(),
					clause.Where{}.Name(),
				)
			}
		}
	}
	return nil
}
