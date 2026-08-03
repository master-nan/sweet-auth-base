\set ON_ERROR_STOP on

-- Development-only Data Permission acceptance data.
-- This script is not part of migrationSteps() or platformSeedSteps().
\if :{?environment}
\else
\set environment ''
\endif

SELECT :'environment' = 'development' AS allow_data_permission_demo_data \gset
\if :allow_data_permission_demo_data
\else
\echo 'Refusing to load Data Permission demo data: pass -v environment=development.'
DO $guard$
BEGIN
    RAISE EXCEPTION 'Data Permission demo data is development-only';
END
$guard$;
\endif

\if :{?app_salt}
\else
\echo 'Refusing to create demo users: pass the current development app salt with -v app_salt=...'
DO $salt_guard$
BEGIN
    RAISE EXCEPTION 'Data Permission demo data requires the development app salt';
END
$salt_guard$;
\endif

\if :{?demo_password}
\else
\set demo_password 'DpDemo@2026'
\endif

BEGIN;

SELECT set_config('sweet.dp_demo.app_salt', :'app_salt', true);
SELECT set_config('sweet.dp_demo.password', :'demo_password', true);

CREATE TABLE IF NOT EXISTS demo_transport_order (
    id bigint PRIMARY KEY,
    order_no varchar(64) NOT NULL UNIQUE,
    owner_org_id bigint NOT NULL,
    amount numeric(18, 2) NOT NULL,
    gmt_create timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    gmt_modify timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    gmt_delete timestamp NULL
);

DO $data_permission_demo$
DECLARE
    v_source_system CONSTANT text := 'sweet_dp_acceptance_v1';
    v_structure_code CONSTANT text := 'DP-ACCEPTANCE-MGMT';
    v_role_marker CONSTANT text := 'DP-5-T001 development acceptance';
    v_now CONSTANT timestamp := CURRENT_TIMESTAMP;

    v_legal_entity_stable_id CONSTANT bigint := 9205601;
    v_structure_stable_id CONSTANT bigint := 9205602;
    v_east_org_stable_id CONSTANT bigint := 9205611;
    v_shanghai_org_stable_id CONSTANT bigint := 9205612;
    v_south_org_stable_id CONSTANT bigint := 9205613;
    v_east_node_stable_id CONSTANT bigint := 9205621;
    v_south_node_stable_id CONSTANT bigint := 9205622;
    v_shanghai_node_stable_id CONSTANT bigint := 9205623;
    v_role_manager_stable_id CONSTANT bigint := 9205001;
    v_role_ungranted_stable_id CONSTANT bigint := 9205002;
    v_user_east_stable_id CONSTANT bigint := 9205101;
    v_user_south_stable_id CONSTANT bigint := 9205102;
    v_user_ungranted_stable_id CONSTANT bigint := 9205103;
    v_employee_east_stable_id CONSTANT bigint := 9205701;
    v_employee_south_stable_id CONSTANT bigint := 9205702;
    v_employee_ungranted_stable_id CONSTANT bigint := 9205703;
    v_assignment_east_stable_id CONSTANT bigint := 9205711;
    v_assignment_south_stable_id CONSTANT bigint := 9205712;
    v_assignment_ungranted_stable_id CONSTANT bigint := 9205713;
    v_table_stable_id CONSTANT bigint := 9205201;
    v_resource_stable_id CONSTANT bigint := 9205301;
    v_ownership_stable_id CONSTANT bigint := 9205321;
    v_policy_stable_id CONSTANT bigint := 9205401;
    v_rule_stable_id CONSTANT bigint := 9205411;

    v_legal_entity_id bigint;
    v_structure_id bigint;
    v_east_org_id bigint;
    v_shanghai_org_id bigint;
    v_south_org_id bigint;
    v_east_node_id bigint;
    v_south_node_id bigint;

    v_user_east_id bigint;
    v_user_south_id bigint;
    v_user_ungranted_id bigint;
    v_role_manager_id bigint;
    v_role_ungranted_id bigint;
    v_employee_east_id bigint;
    v_employee_south_id bigint;
    v_employee_ungranted_id bigint;

    v_table_id bigint;
    v_owner_org_field_id bigint;
    v_dimension_id bigint;
    v_resource_id bigint;
    v_policy_id bigint;
    v_ownership_id bigint;
    v_rule_id bigint;

    v_east_orders text[];
    v_south_orders text[];
BEGIN
    IF EXISTS (
        SELECT 1
        FROM org_structure
        WHERE code = v_structure_code
          AND source_system_code <> v_source_system
    ) THEN
        RAISE EXCEPTION 'Acceptance structure code is already owned by another source';
    END IF;

    INSERT INTO org_legal_entity (
        id, gmt_create, gmt_modify, state, source_system_code, source_id,
        source_code, code, name, short_name, entity_type, status, valid_from,
        source_version, source_updated_at, last_sync_at, source_status,
        source_deleted, sync_status, local_note, local_tags
    )
    VALUES (
        v_legal_entity_stable_id, v_now, v_now, true, v_source_system, 'DP-LE-LOGISTICS',
        'DP-LE-LOGISTICS', 'DP-LE-LOGISTICS', '数据权限验收物流有限公司',
        '验收物流', 'legal_company', 'enabled', CURRENT_DATE - INTERVAL '5 years',
        'dp-demo-v1', v_now, v_now, 'enabled', false, 'success',
        '仅用于开发环境数据权限验收', '["development", "data-permission-acceptance"]'::jsonb
    )
    ON CONFLICT (source_system_code, source_id) DO UPDATE
    SET source_code = EXCLUDED.source_code,
        code = EXCLUDED.code,
        name = EXCLUDED.name,
        short_name = EXCLUDED.short_name,
        entity_type = EXCLUDED.entity_type,
        status = EXCLUDED.status,
        valid_from = EXCLUDED.valid_from,
        valid_to = NULL,
        source_version = EXCLUDED.source_version,
        source_updated_at = EXCLUDED.source_updated_at,
        last_sync_at = EXCLUDED.last_sync_at,
        source_status = EXCLUDED.source_status,
        source_deleted = false,
        sync_status = EXCLUDED.sync_status,
        local_note = EXCLUDED.local_note,
        local_tags = EXCLUDED.local_tags,
        gmt_modify = EXCLUDED.gmt_modify,
        gmt_delete = NULL,
        state = true;

    SELECT id INTO STRICT v_legal_entity_id
    FROM org_legal_entity
    WHERE source_system_code = v_source_system
      AND source_id = 'DP-LE-LOGISTICS';

    INSERT INTO org_structure (
        id, gmt_create, gmt_modify, state, code, name, structure_type,
        source_system_code, source_id, status, is_default, valid_from,
        source_version, last_sync_at, sync_status
    )
    VALUES (
        v_structure_stable_id, v_now, v_now, true, v_structure_code, '数据权限验收管理架构',
        'management', v_source_system, 'DP-STRUCTURE-MGMT', 'enabled', false,
        CURRENT_DATE - INTERVAL '5 years', 'dp-demo-v1', v_now, 'success'
    )
    ON CONFLICT (code) DO UPDATE
    SET name = EXCLUDED.name,
        structure_type = EXCLUDED.structure_type,
        source_system_code = EXCLUDED.source_system_code,
        source_id = EXCLUDED.source_id,
        status = EXCLUDED.status,
        is_default = EXCLUDED.is_default,
        valid_from = EXCLUDED.valid_from,
        valid_to = NULL,
        source_version = EXCLUDED.source_version,
        last_sync_at = EXCLUDED.last_sync_at,
        sync_status = EXCLUDED.sync_status,
        gmt_modify = EXCLUDED.gmt_modify,
        gmt_delete = NULL,
        state = true;

    SELECT id INTO STRICT v_structure_id
    FROM org_structure
    WHERE code = v_structure_code;

    INSERT INTO org_unit (
        id, gmt_create, gmt_modify, state, source_system_code, source_id,
        source_code, code, name, unit_type, primary_legal_entity_id, status,
        valid_from, source_version, source_updated_at, last_sync_at,
        source_status, source_deleted, sync_status, local_note, local_tags,
        display_order
    )
    VALUES
        (
            v_east_org_stable_id, v_now, v_now, true, v_source_system, 'DP-OU-EAST', 'DP-OU-EAST',
            'DP-OU-EAST', '华东物流中心', 'center', v_legal_entity_id, 'enabled',
            CURRENT_DATE - INTERVAL '5 years', 'dp-demo-v1', v_now, v_now,
            'enabled', false, 'success', '仅用于开发环境数据权限验收',
            '["development", "data-permission-acceptance"]'::jsonb, 10
        ),
        (
            v_shanghai_org_stable_id, v_now, v_now, true, v_source_system, 'DP-OU-SHANGHAI',
            'DP-OU-SHANGHAI', 'DP-OU-SHANGHAI', '上海运输部', 'department',
            v_legal_entity_id, 'enabled', CURRENT_DATE - INTERVAL '4 years',
            'dp-demo-v1', v_now, v_now, 'enabled', false, 'success',
            '仅用于开发环境数据权限验收',
            '["development", "data-permission-acceptance"]'::jsonb, 11
        ),
        (
            v_south_org_stable_id, v_now, v_now, true, v_source_system, 'DP-OU-SOUTH', 'DP-OU-SOUTH',
            'DP-OU-SOUTH', '华南物流中心', 'center', v_legal_entity_id, 'enabled',
            CURRENT_DATE - INTERVAL '5 years', 'dp-demo-v1', v_now, v_now,
            'enabled', false, 'success', '仅用于开发环境数据权限验收',
            '["development", "data-permission-acceptance"]'::jsonb, 20
        )
    ON CONFLICT (source_system_code, source_id) DO UPDATE
    SET source_code = EXCLUDED.source_code,
        code = EXCLUDED.code,
        name = EXCLUDED.name,
        unit_type = EXCLUDED.unit_type,
        primary_legal_entity_id = EXCLUDED.primary_legal_entity_id,
        status = EXCLUDED.status,
        valid_from = EXCLUDED.valid_from,
        valid_to = NULL,
        source_version = EXCLUDED.source_version,
        source_updated_at = EXCLUDED.source_updated_at,
        last_sync_at = EXCLUDED.last_sync_at,
        source_status = EXCLUDED.source_status,
        source_deleted = false,
        sync_status = EXCLUDED.sync_status,
        local_note = EXCLUDED.local_note,
        local_tags = EXCLUDED.local_tags,
        display_order = EXCLUDED.display_order,
        gmt_modify = EXCLUDED.gmt_modify,
        gmt_delete = NULL,
        state = true;

    SELECT id INTO STRICT v_east_org_id FROM org_unit
    WHERE source_system_code = v_source_system AND source_id = 'DP-OU-EAST';
    SELECT id INTO STRICT v_shanghai_org_id FROM org_unit
    WHERE source_system_code = v_source_system AND source_id = 'DP-OU-SHANGHAI';
    SELECT id INTO STRICT v_south_org_id FROM org_unit
    WHERE source_system_code = v_source_system AND source_id = 'DP-OU-SOUTH';

    INSERT INTO org_structure_node (
        id, gmt_create, gmt_modify, state, structure_id, org_unit_id,
        parent_node_id, source_system_code, source_id, source_parent_id, path,
        level, sort, valid_from, status, source_deleted, sync_status
    )
    VALUES
        (
            v_east_node_stable_id, v_now, v_now, true, v_structure_id, v_east_org_id, NULL,
            v_source_system, 'DP-NODE-EAST', '', '/pending/DP-NODE-EAST',
            1, 10, CURRENT_DATE - INTERVAL '5 years', 'enabled', false, 'success'
        ),
        (
            v_south_node_stable_id, v_now, v_now, true, v_structure_id, v_south_org_id, NULL,
            v_source_system, 'DP-NODE-SOUTH', '', '/pending/DP-NODE-SOUTH',
            1, 20, CURRENT_DATE - INTERVAL '5 years', 'enabled', false, 'success'
        )
    ON CONFLICT (source_system_code, source_id) DO UPDATE
    SET structure_id = EXCLUDED.structure_id,
        org_unit_id = EXCLUDED.org_unit_id,
        parent_node_id = EXCLUDED.parent_node_id,
        source_parent_id = EXCLUDED.source_parent_id,
        path = EXCLUDED.path,
        level = EXCLUDED.level,
        sort = EXCLUDED.sort,
        valid_from = EXCLUDED.valid_from,
        valid_to = NULL,
        status = EXCLUDED.status,
        source_deleted = false,
        sync_status = EXCLUDED.sync_status,
        gmt_modify = EXCLUDED.gmt_modify,
        gmt_delete = NULL,
        state = true;

    SELECT id INTO STRICT v_east_node_id FROM org_structure_node
    WHERE source_system_code = v_source_system AND source_id = 'DP-NODE-EAST';
    SELECT id INTO STRICT v_south_node_id FROM org_structure_node
    WHERE source_system_code = v_source_system AND source_id = 'DP-NODE-SOUTH';

    UPDATE org_structure_node
    SET path = format('/%s', id)
    WHERE source_system_code = v_source_system
      AND source_id IN ('DP-NODE-EAST', 'DP-NODE-SOUTH');

    INSERT INTO org_structure_node (
        id, gmt_create, gmt_modify, state, structure_id, org_unit_id,
        parent_node_id, source_system_code, source_id, source_parent_id, path,
        level, sort, valid_from, status, source_deleted, sync_status
    )
    VALUES (
        v_shanghai_node_stable_id, v_now, v_now, true, v_structure_id, v_shanghai_org_id, v_east_node_id,
        v_source_system, 'DP-NODE-SHANGHAI', 'DP-NODE-EAST',
        '/pending/DP-NODE-SHANGHAI', 2, 10,
        CURRENT_DATE - INTERVAL '4 years', 'enabled', false, 'success'
    )
    ON CONFLICT (source_system_code, source_id) DO UPDATE
    SET structure_id = EXCLUDED.structure_id,
        org_unit_id = EXCLUDED.org_unit_id,
        parent_node_id = EXCLUDED.parent_node_id,
        source_parent_id = EXCLUDED.source_parent_id,
        path = EXCLUDED.path,
        level = EXCLUDED.level,
        sort = EXCLUDED.sort,
        valid_from = EXCLUDED.valid_from,
        valid_to = NULL,
        status = EXCLUDED.status,
        source_deleted = false,
        sync_status = EXCLUDED.sync_status,
        gmt_modify = EXCLUDED.gmt_modify,
        gmt_delete = NULL,
        state = true;

    UPDATE org_structure_node
    SET path = format('/%s/%s', v_east_node_id, id)
    WHERE source_system_code = v_source_system
      AND source_id = 'DP-NODE-SHANGHAI';

    SELECT id INTO v_role_manager_id
    FROM sys_role
    WHERE name = '物流经理' AND memo = v_role_marker AND gmt_delete IS NULL
    ORDER BY id LIMIT 1;
    IF v_role_manager_id IS NULL THEN
        INSERT INTO sys_role (id, gmt_create, gmt_modify, state, name, memo)
        VALUES (v_role_manager_stable_id, v_now, v_now, true, '物流经理', v_role_marker)
        RETURNING id INTO v_role_manager_id;
    ELSE
        UPDATE sys_role SET state = true, gmt_modify = v_now, gmt_delete = NULL
        WHERE id = v_role_manager_id;
    END IF;

    SELECT id INTO v_role_ungranted_id
    FROM sys_role
    WHERE name = '数据权限验收无授权角色'
      AND memo = v_role_marker
      AND gmt_delete IS NULL
    ORDER BY id LIMIT 1;
    IF v_role_ungranted_id IS NULL THEN
        INSERT INTO sys_role (id, gmt_create, gmt_modify, state, name, memo)
        VALUES (v_role_ungranted_stable_id, v_now, v_now, true, '数据权限验收无授权角色', v_role_marker)
        RETURNING id INTO v_role_ungranted_id;
    ELSE
        UPDATE sys_role SET state = true, gmt_modify = v_now, gmt_delete = NULL
        WHERE id = v_role_ungranted_id;
    END IF;

    INSERT INTO sys_user (
        id, gmt_create, gmt_modify, state, user_name, password, email,
        phone_number, password_changed_at, language, access_tokens, is_reset
    )
    VALUES
        (v_user_east_stable_id, v_now, v_now, true, 'dp_acceptance_east', 'pending',
         'dp-east@example.invalid', '', v_now, 'zh-CN', '', false),
        (v_user_south_stable_id, v_now, v_now, true, 'dp_acceptance_south', 'pending',
         'dp-south@example.invalid', '', v_now, 'zh-CN', '', false),
        (v_user_ungranted_stable_id, v_now, v_now, true, 'dp_acceptance_ungranted', 'pending',
         'dp-ungranted@example.invalid', '', v_now, 'zh-CN', '', false)
    ON CONFLICT (user_name) DO UPDATE
    SET email = EXCLUDED.email,
        phone_number = EXCLUDED.phone_number,
        password_changed_at = EXCLUDED.password_changed_at,
        language = EXCLUDED.language,
        access_tokens = '',
        is_reset = false,
        state = true,
        gmt_modify = EXCLUDED.gmt_modify,
        gmt_delete = NULL;

    SELECT id INTO STRICT v_user_east_id FROM sys_user
    WHERE user_name = 'dp_acceptance_east';
    SELECT id INTO STRICT v_user_south_id FROM sys_user
    WHERE user_name = 'dp_acceptance_south';
    SELECT id INTO STRICT v_user_ungranted_id FROM sys_user
    WHERE user_name = 'dp_acceptance_ungranted';

    UPDATE sys_user
    SET password = md5(
            current_setting('sweet.dp_demo.password') || id::text ||
            current_setting('sweet.dp_demo.app_salt')
        ),
        password_changed_at = v_now,
        gmt_modify = v_now
    WHERE id IN (v_user_east_id, v_user_south_id, v_user_ungranted_id);

    INSERT INTO sys_user_role (user_id, role_id)
    VALUES
        (v_user_east_id, v_role_manager_id),
        (v_user_south_id, v_role_manager_id),
        (v_user_ungranted_id, v_role_ungranted_id)
    ON CONFLICT (user_id, role_id) DO NOTHING;

    IF EXISTS (
        SELECT 1 FROM org_employee
        WHERE user_id IN (v_user_east_id, v_user_south_id, v_user_ungranted_id)
          AND source_system_code <> v_source_system
    ) THEN
        RAISE EXCEPTION 'A demo account is already bound to a non-demo employee';
    END IF;

    INSERT INTO org_employee (
        id, gmt_create, gmt_modify, state, source_system_code, source_id,
        source_code, employee_no, name, employment_status,
        primary_legal_entity_id, valid_from, source_version, source_updated_at,
        last_sync_at, source_deleted, sync_status, user_id, local_note, local_tags
    )
    VALUES
        (
            v_employee_east_stable_id, v_now, v_now, true, v_source_system, 'DP-EMP-EAST', 'DP-EMP-EAST',
            'DP-E1001', '验收用户A', 'active', v_legal_entity_id,
            CURRENT_DATE - INTERVAL '2 years', 'dp-demo-v1', v_now, v_now,
            false, 'success', v_user_east_id, '仅用于开发环境数据权限验收',
            '["development", "data-permission-acceptance"]'::jsonb
        ),
        (
            v_employee_south_stable_id, v_now, v_now, true, v_source_system, 'DP-EMP-SOUTH', 'DP-EMP-SOUTH',
            'DP-E1002', '验收用户B', 'active', v_legal_entity_id,
            CURRENT_DATE - INTERVAL '2 years', 'dp-demo-v1', v_now, v_now,
            false, 'success', v_user_south_id, '仅用于开发环境数据权限验收',
            '["development", "data-permission-acceptance"]'::jsonb
        ),
        (
            v_employee_ungranted_stable_id, v_now, v_now, true, v_source_system, 'DP-EMP-UNGRANTED',
            'DP-EMP-UNGRANTED', 'DP-E1003', '验收无授权用户', 'active',
            v_legal_entity_id, CURRENT_DATE - INTERVAL '2 years', 'dp-demo-v1',
            v_now, v_now, false, 'success', v_user_ungranted_id,
            '仅用于开发环境数据权限验收',
            '["development", "data-permission-acceptance"]'::jsonb
        )
    ON CONFLICT (source_system_code, source_id) DO UPDATE
    SET source_code = EXCLUDED.source_code,
        employee_no = EXCLUDED.employee_no,
        name = EXCLUDED.name,
        employment_status = EXCLUDED.employment_status,
        primary_legal_entity_id = EXCLUDED.primary_legal_entity_id,
        valid_from = EXCLUDED.valid_from,
        valid_to = NULL,
        source_version = EXCLUDED.source_version,
        source_updated_at = EXCLUDED.source_updated_at,
        last_sync_at = EXCLUDED.last_sync_at,
        source_deleted = false,
        sync_status = EXCLUDED.sync_status,
        user_id = EXCLUDED.user_id,
        local_note = EXCLUDED.local_note,
        local_tags = EXCLUDED.local_tags,
        gmt_modify = EXCLUDED.gmt_modify,
        gmt_delete = NULL,
        state = true;

    SELECT id INTO STRICT v_employee_east_id FROM org_employee
    WHERE source_system_code = v_source_system AND source_id = 'DP-EMP-EAST';
    SELECT id INTO STRICT v_employee_south_id FROM org_employee
    WHERE source_system_code = v_source_system AND source_id = 'DP-EMP-SOUTH';
    SELECT id INTO STRICT v_employee_ungranted_id FROM org_employee
    WHERE source_system_code = v_source_system AND source_id = 'DP-EMP-UNGRANTED';

    INSERT INTO org_assignment (
        id, gmt_create, gmt_modify, state, source_system_code, source_id,
        employee_id, legal_entity_id, org_unit_id, assignment_type, is_primary,
        is_manager, valid_from, status, source_version, source_deleted, sync_status
    )
    VALUES
        (
            v_assignment_east_stable_id, v_now, v_now, true, v_source_system, 'DP-ASG-EAST',
            v_employee_east_id, v_legal_entity_id, v_east_org_id, 'standard',
            false, true, CURRENT_DATE - INTERVAL '2 years', 'enabled',
            'dp-demo-v1', false, 'success'
        ),
        (
            v_assignment_south_stable_id, v_now, v_now, true, v_source_system, 'DP-ASG-SOUTH',
            v_employee_south_id, v_legal_entity_id, v_south_org_id, 'standard',
            false, true, CURRENT_DATE - INTERVAL '2 years', 'enabled',
            'dp-demo-v1', false, 'success'
        ),
        (
            v_assignment_ungranted_stable_id, v_now, v_now, true, v_source_system, 'DP-ASG-UNGRANTED',
            v_employee_ungranted_id, v_legal_entity_id, v_east_org_id, 'standard',
            false, false, CURRENT_DATE - INTERVAL '2 years', 'enabled',
            'dp-demo-v1', false, 'success'
        )
    ON CONFLICT (source_system_code, source_id) DO UPDATE
    SET employee_id = EXCLUDED.employee_id,
        legal_entity_id = EXCLUDED.legal_entity_id,
        org_unit_id = EXCLUDED.org_unit_id,
        position_id = NULL,
        assignment_type = EXCLUDED.assignment_type,
        is_primary = EXCLUDED.is_primary,
        is_manager = EXCLUDED.is_manager,
        valid_from = EXCLUDED.valid_from,
        valid_to = NULL,
        status = EXCLUDED.status,
        source_version = EXCLUDED.source_version,
        source_deleted = false,
        sync_status = EXCLUDED.sync_status,
        gmt_modify = EXCLUDED.gmt_modify,
        gmt_delete = NULL,
        state = true;

    INSERT INTO sys_table (
        id, gmt_create, gmt_modify, state, table_name, table_code, table_type,
        parent_id, sql
    )
    VALUES (
        v_table_stable_id, v_now, v_now, true, '数据权限验收运输订单', 'demo_transport_order',
        1, 0, ''
    )
    ON CONFLICT (table_code) DO UPDATE
    SET table_name = EXCLUDED.table_name,
        table_type = EXCLUDED.table_type,
        parent_id = EXCLUDED.parent_id,
        sql = EXCLUDED.sql,
        state = true,
        gmt_modify = EXCLUDED.gmt_modify,
        gmt_delete = NULL;

    SELECT id INTO STRICT v_table_id
    FROM sys_table
    WHERE table_code = 'demo_transport_order';

    INSERT INTO sys_table_field (
        id, gmt_create, gmt_modify, state, table_id, field_name, field_code,
        field_type, field_length, field_decimal_length, input_type,
        is_primary_key, is_index, is_quick_search, is_advanced_search,
        is_sort, is_null, is_list_show, is_insert_show, is_update_show,
        sequence, original_field_id, binding, field_category
    )
    VALUES
        (9205211, v_now, v_now, true, v_table_id, 'ID', 'id', 1, 0, 0, 2,
         true, true, false, false, true, false, true, false, false,
         1, 0, '', 'normal_field'),
        (9205212, v_now, v_now, true, v_table_id, '运单号', 'order_no', 3, 64, 0, 1,
         false, true, true, true, true, false, true, true, true,
         2, 0, 'required', 'normal_field'),
        (9205213, v_now, v_now, true, v_table_id, '所属管理组织', 'owner_org_id', 1, 0, 0, 2,
         false, true, false, true, true, false, true, true, false,
         3, 0, 'required', 'normal_field'),
        (9205214, v_now, v_now, true, v_table_id, '金额', 'amount', 2, 18, 2, 2,
         false, false, false, true, true, false, true, true, true,
         4, 0, 'required', 'normal_field'),
        (9205215, v_now, v_now, true, v_table_id, '删除时间', 'gmt_delete', 7, 0, 0, 6,
         false, false, false, false, false, true, false, false, false,
         5, 0, '', 'normal_field')
    ON CONFLICT (table_id, field_code) DO UPDATE
    SET field_name = EXCLUDED.field_name,
        field_type = EXCLUDED.field_type,
        field_length = EXCLUDED.field_length,
        field_decimal_length = EXCLUDED.field_decimal_length,
        input_type = EXCLUDED.input_type,
        is_primary_key = EXCLUDED.is_primary_key,
        is_index = EXCLUDED.is_index,
        is_quick_search = EXCLUDED.is_quick_search,
        is_advanced_search = EXCLUDED.is_advanced_search,
        is_sort = EXCLUDED.is_sort,
        is_null = EXCLUDED.is_null,
        is_list_show = EXCLUDED.is_list_show,
        is_insert_show = EXCLUDED.is_insert_show,
        is_update_show = EXCLUDED.is_update_show,
        sequence = EXCLUDED.sequence,
        binding = EXCLUDED.binding,
        field_category = EXCLUDED.field_category,
        expression = NULL,
        gmt_modify = EXCLUDED.gmt_modify,
        gmt_delete = NULL,
        state = true;

    SELECT id INTO STRICT v_owner_org_field_id
    FROM sys_table_field
    WHERE table_id = v_table_id
      AND field_code = 'owner_org_id'
      AND gmt_delete IS NULL;

    SELECT id INTO v_dimension_id
    FROM sys_data_dimension_definition
    WHERE code = 'management_org'
      AND state = true
      AND gmt_delete IS NULL;
    IF v_dimension_id IS NULL THEN
        RAISE EXCEPTION 'Required management_org dimension seed is missing';
    END IF;

    IF EXISTS (
        SELECT 1 FROM sys_data_resource
        WHERE resource_code = 'transport_order'
          AND (
              resource_type <> 'low_code_table'
              OR table_id IS DISTINCT FROM v_table_id
          )
    ) THEN
        RAISE EXCEPTION 'Resource code transport_order is already bound to another target';
    END IF;

    INSERT INTO sys_data_resource (
        id, gmt_create, gmt_modify, state, resource_code, name, resource_type,
        table_id, adapter_code, permission_enabled, description
    )
    VALUES (
        v_resource_stable_id, v_now, v_now, true, 'transport_order', '运输订单', 'low_code_table',
        v_table_id, 'metadata_filter', true, '仅用于开发环境数据权限验收'
    )
    ON CONFLICT (resource_code) DO UPDATE
    SET name = EXCLUDED.name,
        resource_type = EXCLUDED.resource_type,
        table_id = EXCLUDED.table_id,
        service_code = NULL,
        report_definition_id = NULL,
        adapter_code = EXCLUDED.adapter_code,
        permission_enabled = true,
        description = EXCLUDED.description,
        state = true,
        gmt_modify = EXCLUDED.gmt_modify,
        gmt_delete = NULL;

    SELECT id INTO STRICT v_resource_id
    FROM sys_data_resource
    WHERE resource_code = 'transport_order';

    INSERT INTO sys_data_resource_operation (
        id, gmt_create, gmt_modify, state, resource_id, operation,
        permission_enabled, description
    )
    VALUES
        (9205311, v_now, v_now, true, v_resource_id, 'query', true,
         '数据权限验收列表与总数查询'),
        (9205312, v_now, v_now, true, v_resource_id, 'detail', true,
         '数据权限验收详情查询')
    ON CONFLICT (resource_id, operation) WHERE gmt_delete IS NULL DO UPDATE
    SET permission_enabled = true,
        description = EXCLUDED.description,
        state = true,
        gmt_modify = EXCLUDED.gmt_modify,
        gmt_delete = NULL;

    SELECT id INTO v_ownership_id
    FROM sys_data_ownership_field
    WHERE resource_id = v_resource_id
      AND ownership_code = 'owner_org'
      AND gmt_delete IS NULL;
    IF v_ownership_id IS NULL THEN
        INSERT INTO sys_data_ownership_field (
            id, gmt_create, gmt_modify, state, resource_id, ownership_code,
            dimension_id, binding_type, table_field_id, adapter_field_code,
            value_type, description
        )
        VALUES (
            v_ownership_stable_id, v_now, v_now, true, v_resource_id, 'owner_org', v_dimension_id,
            'metadata_field', v_owner_org_field_id, NULL, 'bigint',
            '运输订单所属管理组织'
        )
        RETURNING id INTO v_ownership_id;
    ELSE
        UPDATE sys_data_ownership_field
        SET dimension_id = v_dimension_id,
            binding_type = 'metadata_field',
            table_field_id = v_owner_org_field_id,
            adapter_field_code = NULL,
            value_type = 'bigint',
            description = '运输订单所属管理组织',
            state = true,
            gmt_modify = v_now,
            gmt_delete = NULL
        WHERE id = v_ownership_id;
    END IF;

    INSERT INTO sys_data_policy (
        id, gmt_create, gmt_modify, state, code, name, policy_type, description
    )
    VALUES (
        v_policy_stable_id, v_now, v_now, true, 'dp_acceptance_org_descendants',
        '本组织及下级组织', 'rule_set', '仅用于开发环境数据权限验收'
    )
    ON CONFLICT (code) DO UPDATE
    SET name = EXCLUDED.name,
        policy_type = EXCLUDED.policy_type,
        description = EXCLUDED.description,
        state = true,
        gmt_modify = EXCLUDED.gmt_modify,
        gmt_delete = NULL;

    SELECT id INTO STRICT v_policy_id
    FROM sys_data_policy
    WHERE code = 'dp_acceptance_org_descendants';

    SELECT id INTO v_rule_id
    FROM sys_data_policy_rule
    WHERE policy_id = v_policy_id
      AND sequence = 1
      AND gmt_delete IS NULL;
    IF v_rule_id IS NULL THEN
        INSERT INTO sys_data_policy_rule (
            id, gmt_create, gmt_modify, state, policy_id, sequence, dimension_id,
            ownership_code, scope_source, relation, operator,
            specified_values, structure_code, description
        )
        VALUES (
            v_rule_stable_id, v_now, v_now, true, v_policy_id, 1, v_dimension_id,
            'owner_org', 'effective_org_units', 'self_and_descendants', 'in',
            NULL, v_structure_code, '当前有效管理组织及其下级组织'
        )
        RETURNING id INTO v_rule_id;
    ELSE
        UPDATE sys_data_policy_rule
        SET dimension_id = v_dimension_id,
            ownership_code = 'owner_org',
            scope_source = 'effective_org_units',
            relation = 'self_and_descendants',
            operator = 'in',
            specified_values = NULL,
            structure_code = v_structure_code,
            description = '当前有效管理组织及其下级组织',
            state = true,
            gmt_modify = v_now,
            gmt_delete = NULL
        WHERE id = v_rule_id;
    END IF;

    INSERT INTO sys_data_grant (
        id, gmt_create, gmt_modify, state, subject_type, subject_id, resource_id,
        operation, policy_id, description
    )
    VALUES
        (9205501, v_now, v_now, true, 'role', v_role_manager_id, v_resource_id,
         'query', v_policy_id, '物流经理运输订单查询范围'),
        (9205502, v_now, v_now, true, 'role', v_role_manager_id, v_resource_id,
         'detail', v_policy_id, '物流经理运输订单详情范围')
    ON CONFLICT (
        subject_type, subject_id, resource_id, operation, policy_id
    ) WHERE gmt_delete IS NULL DO UPDATE
    SET state = true,
        description = EXCLUDED.description,
        valid_from = NULL,
        valid_to = NULL,
        gmt_modify = EXCLUDED.gmt_modify,
        gmt_delete = NULL;

    INSERT INTO demo_transport_order (
        id, order_no, owner_org_id, amount, gmt_create, gmt_modify, gmt_delete
    )
    VALUES
        (910001, 'ORD001', v_east_org_id, 1000.00, v_now, v_now, NULL),
        (910002, 'ORD002', v_shanghai_org_id, 2000.00, v_now, v_now, NULL),
        (910003, 'ORD003', v_south_org_id, 3000.00, v_now, v_now, NULL)
    ON CONFLICT (order_no) DO UPDATE
    SET owner_org_id = EXCLUDED.owner_org_id,
        amount = EXCLUDED.amount,
        gmt_modify = EXCLUDED.gmt_modify,
        gmt_delete = NULL;

    IF (
        SELECT count(*) FROM org_unit
        WHERE source_system_code = v_source_system
          AND source_id IN ('DP-OU-EAST', 'DP-OU-SHANGHAI', 'DP-OU-SOUTH')
          AND state = true
          AND status = 'enabled'
          AND source_deleted = false
          AND gmt_delete IS NULL
    ) <> 3 THEN
        RAISE EXCEPTION 'Organization acceptance data is incomplete';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM org_structure_node child
        JOIN org_structure_node parent ON parent.id = child.parent_node_id
        WHERE child.structure_id = v_structure_id
          AND child.org_unit_id = v_shanghai_org_id
          AND parent.org_unit_id = v_east_org_id
          AND child.state = true
          AND parent.state = true
          AND child.gmt_delete IS NULL
          AND parent.gmt_delete IS NULL
    ) THEN
        RAISE EXCEPTION 'Shanghai transport department is not a child of East logistics center';
    END IF;

    IF (
        SELECT count(*)
        FROM org_employee
        WHERE source_system_code = v_source_system
          AND user_id IN (v_user_east_id, v_user_south_id, v_user_ungranted_id)
          AND gmt_delete IS NULL
    ) <> 3 THEN
        RAISE EXCEPTION 'Demo account bindings are incomplete';
    END IF;

    IF EXISTS (
        SELECT 1 FROM sys_data_grant
        WHERE subject_type = 'role'
          AND subject_id = v_role_ungranted_id
          AND resource_id = v_resource_id
          AND state = true
          AND gmt_delete IS NULL
    ) THEN
        RAISE EXCEPTION 'The ungranted acceptance role unexpectedly has a Grant';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM sys_data_resource resource
        JOIN sys_data_ownership_field ownership
          ON ownership.resource_id = resource.id
         AND ownership.gmt_delete IS NULL
        JOIN sys_data_policy_rule rule
          ON rule.policy_id = v_policy_id
         AND rule.gmt_delete IS NULL
        WHERE resource.id = v_resource_id
          AND resource.permission_enabled = true
          AND resource.table_id = v_table_id
          AND ownership.ownership_code = rule.ownership_code
          AND ownership.dimension_id = rule.dimension_id
          AND ownership.binding_type = 'metadata_field'
          AND ownership.table_field_id = v_owner_org_field_id
          AND rule.scope_source = 'effective_org_units'
          AND rule.relation = 'self_and_descendants'
          AND rule.operator = 'in'
          AND rule.structure_code = v_structure_code
          AND resource.state = true
          AND ownership.state = true
          AND rule.state = true
    ) THEN
        RAISE EXCEPTION 'Data Permission acceptance configuration is inconsistent';
    END IF;

    WITH RECURSIVE east_scope AS (
        SELECT node.id, node.org_unit_id
        FROM org_structure_node node
        WHERE node.structure_id = v_structure_id
          AND node.org_unit_id = v_east_org_id
          AND node.state = true
          AND node.status = 'enabled'
          AND node.source_deleted = false
          AND node.gmt_delete IS NULL
        UNION ALL
        SELECT child.id, child.org_unit_id
        FROM org_structure_node child
        JOIN east_scope parent ON parent.id = child.parent_node_id
        WHERE child.structure_id = v_structure_id
          AND child.state = true
          AND child.status = 'enabled'
          AND child.source_deleted = false
          AND child.gmt_delete IS NULL
    )
    SELECT array_agg(order_no ORDER BY order_no)
    INTO v_east_orders
    FROM demo_transport_order
    WHERE owner_org_id IN (SELECT org_unit_id FROM east_scope)
      AND gmt_delete IS NULL;

    WITH RECURSIVE south_scope AS (
        SELECT node.id, node.org_unit_id
        FROM org_structure_node node
        WHERE node.structure_id = v_structure_id
          AND node.org_unit_id = v_south_org_id
          AND node.state = true
          AND node.status = 'enabled'
          AND node.source_deleted = false
          AND node.gmt_delete IS NULL
        UNION ALL
        SELECT child.id, child.org_unit_id
        FROM org_structure_node child
        JOIN south_scope parent ON parent.id = child.parent_node_id
        WHERE child.structure_id = v_structure_id
          AND child.state = true
          AND child.status = 'enabled'
          AND child.source_deleted = false
          AND child.gmt_delete IS NULL
    )
    SELECT array_agg(order_no ORDER BY order_no)
    INTO v_south_orders
    FROM demo_transport_order
    WHERE owner_org_id IN (SELECT org_unit_id FROM south_scope)
      AND gmt_delete IS NULL;

    IF v_east_orders IS DISTINCT FROM ARRAY['ORD001', 'ORD002']::text[] THEN
        RAISE EXCEPTION 'East acceptance data resolved %, expected ORD001 and ORD002', v_east_orders;
    END IF;
    IF v_south_orders IS DISTINCT FROM ARRAY['ORD003']::text[] THEN
        RAISE EXCEPTION 'South acceptance data resolved %, expected ORD003', v_south_orders;
    END IF;

    RAISE NOTICE 'Data Permission acceptance data is ready: users %, %, %',
        v_user_east_id, v_user_south_id, v_user_ungranted_id;
END
$data_permission_demo$;

COMMIT;
