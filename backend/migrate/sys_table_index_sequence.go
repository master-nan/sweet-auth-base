package main

import "gorm.io/gorm"

// backfillSysTableIndexFieldSequence 将旧Metadata顺序对齐PostgreSQL复合索引的物理顺序；
// 新写入会直接持久化sequence。
func backfillSysTableIndexFieldSequence(db *gorm.DB) error {
	if db.Dialector.Name() != "postgres" {
		return nil
	}
	return db.Exec(`
WITH physical_index_fields AS (
    SELECT
        sti.id AS index_id,
        stf.id AS field_id,
        key.ordinal_position AS sequence
    FROM sys_table_index sti
    JOIN sys_table st ON st.id = sti.table_id AND st.gmt_delete IS NULL
    JOIN pg_namespace ns ON ns.nspname = current_schema()
    JOIN pg_class tbl ON tbl.relnamespace = ns.oid AND tbl.relname = st.table_code
    JOIN pg_index pix ON pix.indrelid = tbl.oid
    JOIN pg_class idx ON idx.oid = pix.indexrelid AND idx.relname = sti.index_name
    CROSS JOIN LATERAL unnest(pix.indkey) WITH ORDINALITY AS key(attnum, ordinal_position)
    JOIN pg_attribute attr ON attr.attrelid = tbl.oid AND attr.attnum = key.attnum
    JOIN sys_table_field stf
      ON stf.table_id = st.id
     AND stf.field_code = attr.attname
     AND stf.gmt_delete IS NULL
)
UPDATE sys_table_index_field sif
SET sequence = physical.sequence
FROM physical_index_fields physical
WHERE sif.index_id = physical.index_id
  AND sif.field_id = physical.field_id
  AND sif.sequence IS DISTINCT FROM physical.sequence`).Error
}
