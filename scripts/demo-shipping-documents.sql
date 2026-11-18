-- 本地演示：在已有“TMS测试”数据集上按需安装，不作为生产迁移。
-- docker exec -i sweet-auth-base-postgres-1 psql -X -U sweet_admin -d sweet_admin < scripts/demo-shipping-documents.sql
\set ON_ERROR_STOP on
BEGIN;
SELECT pg_advisory_xact_lock(890610000);
SELECT EXISTS (SELECT 1 FROM sys_menu WHERE id=890610101 AND table_code='demo_ship_log'
               AND name='lowcode_demo_ship_log') AS installed \gset
SELECT EXISTS (SELECT 1 FROM sys_menu WHERE id=890610100 AND name='shipping_doc_lines'
               AND title='单据明细表' AND NOT is_hidden AND gmt_delete IS NULL) AS old_navigation \gset
SELECT NOT :'installed'::boolean OR :'old_navigation'::boolean AS install_navigation \gset
\if :install_navigation
-- 隐藏的是目录而不是子表授权入口。单据页签仍按子表菜单、按钮和数据权限读写。
INSERT INTO sys_menu (id,state,pid,name,path,component,title,sequence,page_type,icon,is_hidden) VALUES
(890610107,true,0,'export_shipping','export-shipping','src/components/Layout/Layout.vue','外销船运',92,'directory','directions_boat',false),
(890610108,true,890610107,'ship_ledger_lines','ledger-lines','src/components/Layout/Layout.vue','台账明细',2,'directory','folder',true);
INSERT INTO sys_role_menu (role_id,menu_id)
SELECT r.id,m.id FROM sys_role r CROSS JOIN sys_menu m WHERE r.name='super_admin' AND m.id IN (890610107,890610108);
\endif
\if :installed
\if :old_navigation
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM sys_menu WHERE pid=890610100 AND gmt_delete IS NULL
             AND id NOT IN (890610102,890610103,890610104,890610105,890610106))
  THEN RAISE EXCEPTION '旧演示目录包含其他菜单，请先移出后再整理'; END IF;
END $$;
UPDATE sys_menu SET title='运单明细',is_hidden=true WHERE id=890610100;
UPDATE sys_menu SET pid=890610107 WHERE id=890610101
  AND pid=(SELECT pid FROM sys_menu WHERE name='lowcode_tms_waybill');
UPDATE sys_menu SET pid=890610108 WHERE id IN (890610102,890610103) AND pid=890610100;
\endif
\echo 保留已安装的单据和明细。
\else
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM sys_menu WHERE name='lowcode_tms_waybill' AND state AND gmt_delete IS NULL)
     OR NOT EXISTS (SELECT 1 FROM sys_role WHERE name='super_admin' AND state AND gmt_delete IS NULL)
     OR NOT EXISTS (SELECT 1 FROM tms_company WHERE state AND gmt_delete IS NULL)
     OR (SELECT count(DISTINCT event_action) FROM sys_menu_button_template WHERE scene='lowcode_crud'
          AND state AND gmt_delete IS NULL AND event_action IN ('query','detail','create','update','delete')) <> 5
  THEN RAISE EXCEPTION '请先初始化 TMS 演示、管理员角色和通用页面按钮模板'; END IF;
END $$;

CREATE TEMP TABLE demo_base (
  id bigint NOT NULL, gmt_create timestamptz NOT NULL DEFAULT now(), gmt_create_user bigint NOT NULL,
  gmt_modify timestamptz NOT NULL DEFAULT now(), gmt_modify_user bigint NOT NULL,
  gmt_delete timestamptz, gmt_delete_user bigint, state boolean NOT NULL DEFAULT true
) ON COMMIT DROP;
CREATE TABLE demo_ship_log (
  LIKE demo_base INCLUDING DEFAULTS, PRIMARY KEY(id), ledger_no varchar(64) NOT NULL UNIQUE,
  booking_date date NOT NULL, vessel varchar(64) NOT NULL, cargo varchar(64), origin_port varchar(64),
  dest_port varchar(64), supplier varchar(64), settlement_company varchar(64),
  planned_tons numeric(14,3), loaded_tons numeric(14,3) NOT NULL CHECK (loaded_tons >= 0),
  unit_rate numeric(14,2) NOT NULL CHECK (unit_rate >= 0), delay_days integer NOT NULL DEFAULT 0 CHECK (delay_days >= 0),
  delay_rate numeric(10,2) NOT NULL DEFAULT 1 CHECK (delay_rate >= 0),
  freight numeric(18,2) GENERATED ALWAYS AS (round(loaded_tons * unit_rate,2)) STORED,
  delay_fee numeric(18,2) GENERATED ALWAYS AS (round(loaded_tons * delay_days * delay_rate,2)) STORED,
  payable numeric(18,2) GENERATED ALWAYS AS (round(loaded_tons * unit_rate,2) + round(loaded_tons * delay_days * delay_rate,2)) STORED
);
CREATE TABLE demo_ship_pay (
  LIKE demo_base INCLUDING DEFAULTS, PRIMARY KEY(id), ledger_id bigint NOT NULL REFERENCES demo_ship_log,
  paid_date date NOT NULL, pay_kind varchar(64) NOT NULL, voucher varchar(64), paid_amount numeric(18,2) NOT NULL CHECK (paid_amount > 0)
);
CREATE TABLE demo_ship_bill (
  LIKE demo_base INCLUDING DEFAULTS, PRIMARY KEY(id), ledger_id bigint NOT NULL REFERENCES demo_ship_log,
  received_date date NOT NULL, invoice_no varchar(64) NOT NULL, issuer varchar(64), invoice_amount numeric(18,2) NOT NULL CHECK (invoice_amount > 0)
);
CREATE TABLE demo_way_goods (
  LIKE demo_base INCLUDING DEFAULTS, PRIMARY KEY(id), waybill_id bigint NOT NULL REFERENCES tms_waybill,
  goods_name varchar(64) NOT NULL, specification varchar(64), pieces integer NOT NULL CHECK (pieces > 0)
);
CREATE TABLE demo_way_income (
  LIKE demo_base INCLUDING DEFAULTS, PRIMARY KEY(id), waybill_id bigint NOT NULL REFERENCES tms_waybill,
  income_item varchar(64) NOT NULL, payer varchar(64), income_amount numeric(18,2) NOT NULL CHECK (income_amount > 0)
);
CREATE TABLE demo_way_cost (
  LIKE demo_base INCLUDING DEFAULTS, PRIMARY KEY(id), waybill_id bigint NOT NULL REFERENCES tms_waybill,
  cost_item varchar(64) NOT NULL, payee varchar(64), cost_amount numeric(18,2) NOT NULL CHECK (cost_amount > 0)
);

CREATE TEMP TABLE demo_catalog (n int PRIMARY KEY, code text, title text, labels text[]) ON COMMIT DROP;
INSERT INTO demo_catalog VALUES
(1,'demo_ship_log','外销船运台账',ARRAY['台账号','订船日期','船号','货物名称','起运港','目的港','供应商','结算主体','计划吨位','实装吨位','运费单价（元/吨）','滞期天数','滞期费率（元/吨/天）','运费（元）','滞期费（元）','应付总额（元）']),
(2,'demo_ship_pay','实际付款',ARRAY['所属台账','实付日期','款项类型','付款凭证号','实付金额（元）']),
(3,'demo_ship_bill','发票到票',ARRAY['所属台账','到票日期','发票号码','开票方','价税合计（元）']),
(4,'demo_way_goods','货物明细',ARRAY['所属运单','货物名称','规格','数量（件）']),
(5,'demo_way_income','收入明细',ARRAY['所属运单','收入项目','付款方','收入金额（元）']),
(6,'demo_way_cost','费用明细',ARRAY['所属运单','费用项目','收款方','费用金额（元）']);

INSERT INTO sys_table (id,state,gmt_create,gmt_modify,table_name,table_code,table_type,form_open_mode,detail_open_mode)
SELECT 890610000+n,true,now(),now(),title,code,1,'page','page' FROM demo_catalog;
-- 系统字段沿用当前 TMS 元数据，业务字段取实体列的精度和可空属性。
INSERT INTO sys_table_field (id,state,table_id,field_name,field_code,field_type,input_type,sequence,is_null,
  is_primary_key,is_list_show,is_insert_show,is_update_show,dict_code,field_category)
SELECT 890620000+c.n*100+col.ordinal_position,true,890610000+c.n,f.field_name,f.field_code,f.field_type,f.input_type,
  f.sequence,f.is_null,f.is_primary_key,false,false,false,f.dict_code,f.field_category
FROM demo_catalog c JOIN information_schema.columns col ON col.table_schema=current_schema() AND col.table_name=c.code
JOIN sys_table_field f ON f.table_id=(SELECT id FROM sys_table WHERE table_code='tms_waybill')
  AND f.field_code=col.column_name AND f.gmt_delete IS NULL WHERE col.ordinal_position<=8;
INSERT INTO sys_table_field (id,state,table_id,field_name,field_code,field_type,input_type,sequence,field_length,
  field_decimal_length,numeric_precision,numeric_scale,logical_type,display_format,is_null,is_primary_key,
  is_list_show,is_insert_show,is_update_show,is_quick_search,is_sort,field_category,default_value,linkage_config)
SELECT 890620000+c.n*100+p,true,890610000+c.n,labels[p-8],column_name,
  CASE data_type WHEN 'bigint' THEN 1 WHEN 'numeric' THEN 2 WHEN 'date' THEN 6 WHEN 'integer' THEN 11 ELSE 3 END,
  CASE data_type WHEN 'bigint' THEN 4 WHEN 'numeric' THEN 2 WHEN 'integer' THEN 2 WHEN 'date' THEN 5 ELSE 1 END,
  p,coalesce(character_maximum_length,0),CASE WHEN data_type='numeric' THEN numeric_scale ELSE 0 END,
  CASE WHEN data_type='numeric' THEN numeric_precision ELSE 0 END,CASE WHEN data_type='numeric' THEN numeric_scale ELSE 0 END,
  fmt,fmt,is_nullable='YES',false,
  (c.n<>1 OR column_name IN ('ledger_no','vessel','origin_port','dest_port','loaded_tons','freight','delay_fee','payable')),
  is_generated='NEVER',is_generated='NEVER' AND data_type<>'bigint',
  column_name IN ('ledger_no','vessel','goods_name','invoice_no'),true,'normal_field',
  CASE column_name WHEN 'delay_days' THEN '0' WHEN 'delay_rate' THEN '1' END,
  CASE WHEN column_name IN ('ledger_id','waybill_id') THEN jsonb_build_object('linkage', jsonb_build_object(
    'enabled',true,'mode','relation','tableCode',CASE WHEN c.n<4 THEN 'demo_ship_log' ELSE 'tms_waybill' END,
    'labelKey',CASE WHEN c.n<4 THEN 'ledger_no' ELSE 'waybill_no' END,'valueKey','id','pageSize',200))::text END
FROM demo_catalog c JOIN information_schema.columns col ON col.table_schema=current_schema() AND col.table_name=c.code
CROSS JOIN LATERAL (SELECT col.ordinal_position AS p,CASE
  WHEN data_type='numeric' AND column_name NOT IN ('planned_tons','loaded_tons') THEN 'money'
  WHEN data_type='numeric' THEN 'decimal' WHEN data_type='integer' THEN 'integer'
  WHEN data_type='date' THEN 'date' ELSE 'plain' END AS fmt) flags WHERE p>8;

INSERT INTO sys_table_relation (id,state,table_id,related_table_id,reference_key,foreign_key,relation_type,
  document_tab,quantity_field,amount_field)
SELECT 890630000+n,true,CASE WHEN n<4 THEN 890610001 ELSE (SELECT id FROM sys_table WHERE table_code='tms_waybill') END,
  890610000+n,'id',CASE WHEN n<4 THEN 'ledger_id' ELSE 'waybill_id' END,'2',title,
  CASE WHEN n=4 THEN 'pieces' ELSE '' END,
  CASE n WHEN 2 THEN 'paid_amount' WHEN 3 THEN 'invoice_amount' WHEN 5 THEN 'income_amount' WHEN 6 THEN 'cost_amount' ELSE '' END
FROM demo_catalog WHERE n>1;
DO $$ DECLARE item record; BEGIN
  FOR item IN SELECT * FROM demo_catalog WHERE n>1 LOOP
    EXECUTE format('CREATE INDEX ON %I (%I)',item.code,CASE WHEN item.n<4 THEN 'ledger_id' ELSE 'waybill_id' END);
  END LOOP;
END $$;

INSERT INTO sys_menu (id,state,pid,name,path,component,title,sequence,page_type,icon,is_hidden)
SELECT 890610100,true,pid,'shipping_doc_lines','document-lines','src/components/Layout/Layout.vue',
  '运单明细',91,'directory','folder',true FROM sys_menu WHERE name='lowcode_tms_waybill';
INSERT INTO sys_menu (id,state,pid,name,path,component,title,sequence,page_type,table_code,option,icon,is_hidden)
SELECT 890610100+n,true,CASE WHEN n=1 THEN 890610107 WHEN n IN (2,3) THEN 890610108 ELSE 890610100 END,
  'lowcode_'||code,CASE WHEN n=1 THEN 'shipping-ledger' ELSE code END,'pages/develop/generalization/Index.vue',
  title,CASE WHEN n=1 THEN 90 ELSE n END,'low_code',code,'',
  CASE WHEN n=1 THEN 'directions_boat' ELSE 'table_rows' END,false FROM demo_catalog;
INSERT INTO sys_menu_button (id,state,menu_id,name,code,memo,position,event_type,event_action,icon,color,display_mode,sequence,
  path,method,params_schema,confirm_text,disable_when,is_button,is_hidden,is_disabled,before_hooks,after_hooks)
SELECT 890640000+c.n*100+row_number() OVER (PARTITION BY c.n ORDER BY b.id),true,890610100+c.n,b.name,
  c.code||b.code_suffix,b.memo,b.position,b.event_type,b.event_action,b.icon,b.color,b.display_mode,b.sequence,
  b.path,b.method,b.params_schema,b.confirm_text,b.disable_when,b.is_button,false,b.is_disabled,b.before_hooks,b.after_hooks
FROM demo_catalog c CROSS JOIN sys_menu_button_template b WHERE b.scene='lowcode_crud' AND b.state AND b.gmt_delete IS NULL;
INSERT INTO sys_role_menu (role_id,menu_id)
SELECT r.id,m.id FROM sys_role r CROSS JOIN sys_menu m WHERE r.name='super_admin' AND m.id BETWEEN 890610100 AND 890610106;
INSERT INTO sys_role_menu_button (role_id,menu_id,button_id)
SELECT r.id,b.menu_id,b.id FROM sys_role r CROSS JOIN sys_menu_button b WHERE r.name='super_admin' AND b.menu_id BETWEEN 890610101 AND 890610106;

INSERT INTO demo_ship_log (id,gmt_create_user,gmt_modify_user,ledger_no,booking_date,vessel,cargo,origin_port,dest_port,
  supplier,settlement_company,planned_tons,loaded_tons,unit_rate,delay_days,delay_rate)
VALUES (890650001,0,0,'DEMO-SHIP-001',current_date,'演示货轮01','碎石','示例起运港','示例目的港','示例船务公司','示例结算公司',1050,1000,20,1,1);
INSERT INTO demo_ship_pay (id,gmt_create_user,gmt_modify_user,ledger_id,paid_date,pay_kind,voucher,paid_amount)
VALUES (890650011,0,0,890650001,current_date,'定金','DEMO-PAY-01',5000),(890650012,0,0,890650001,current_date,'进度款','DEMO-PAY-02',7000);
INSERT INTO demo_ship_bill (id,gmt_create_user,gmt_modify_user,ledger_id,received_date,invoice_no,issuer,invoice_amount)
VALUES (890650021,0,0,890650001,current_date,'DEMO-INV-01','示例船务公司',9000);
INSERT INTO tms_waybill (id,gmt_create,gmt_create_user,gmt_modify,gmt_modify_user,state,waybill_no,company_id,customer_name,status)
SELECT 890650002,now(),0,now(),0,true,'DEMO-WB-001',id,'演示客户（主子表）','已创建' FROM tms_company
WHERE state AND gmt_delete IS NULL ORDER BY id LIMIT 1;
INSERT INTO demo_way_goods (id,gmt_create_user,gmt_modify_user,waybill_id,goods_name,specification,pieces)
VALUES (890650031,0,0,890650002,'演示配件','箱装',12),(890650032,0,0,890650002,'演示耗材','箱装',8);
INSERT INTO demo_way_income (id,gmt_create_user,gmt_modify_user,waybill_id,income_item,payer,income_amount)
VALUES (890650041,0,0,890650002,'运输费','演示客户',3200),(890650042,0,0,890650002,'装卸服务','演示客户',800);
INSERT INTO demo_way_cost (id,gmt_create_user,gmt_modify_user,waybill_id,cost_item,payee,cost_amount)
VALUES (890650051,0,0,890650002,'承运费','演示承运商',1200),(890650052,0,0,890650002,'装卸费','演示装卸队',300);
\endif
COMMIT;
