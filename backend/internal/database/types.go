/**
 * @Author: Nan
 * @Date: 2024/10/17 16:24
 */

package database

import "gorm.io/gorm"

type PrimaryDB struct {
	DB *gorm.DB
}
