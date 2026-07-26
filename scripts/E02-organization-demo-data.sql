\set ON_ERROR_STOP on

-- Development-only Organization Foundation acceptance data.
-- This file is intentionally not registered by migrationSteps() or platformSeedSteps().
\if :{?environment}
\else
\set environment ''
\endif

SELECT :'environment' = 'development' AS allow_organization_demo_data \gset
\if :allow_organization_demo_data
\else
\echo 'Refusing to load Organization demo data: pass -v environment=development.'
DO $guard$
BEGIN
    RAISE EXCEPTION 'Organization demo data is development-only';
END
$guard$;
\endif

BEGIN;

DO $organization_demo$
DECLARE
    v_source_system CONSTANT text := 'sweet_dev_org_acceptance_v1';
    v_now CONSTANT timestamp := CURRENT_TIMESTAMP;

    v_admin_user_id bigint;

    v_legal_group_id bigint;
    v_legal_cn_id bigint;
    v_legal_logistics_id bigint;
    v_legal_overseas_id bigint;

    v_structure_admin_id bigint;
    v_structure_region_id bigint;

    v_unit_hq_id bigint;
    v_unit_east_id bigint;
    v_unit_south_id bigint;
    v_unit_sales_id bigint;
    v_unit_ops_id bigint;
    v_unit_shared_id bigint;
    v_unit_project_id bigint;
    v_unit_east_region_id bigint;
    v_unit_south_region_id bigint;

    v_admin_hq_node_id bigint;
    v_admin_east_node_id bigint;
    v_admin_south_node_id bigint;
    v_region_east_node_id bigint;
    v_region_south_node_id bigint;

    v_position_sales_manager_id bigint;
    v_position_sales_specialist_id bigint;
    v_position_shared_finance_id bigint;
    v_position_ops_id bigint;
    v_position_project_id bigint;
    v_position_legacy_id bigint;

    v_employee_zhang_id bigint;
    v_employee_li_id bigint;
    v_employee_wang_id bigint;
    v_employee_history_id bigint;
    v_employee_future_id bigint;

    v_batch_success_id bigint;
    v_batch_failed_id bigint;
    v_sync_record_id bigint;
BEGIN
    SELECT id
    INTO v_admin_user_id
    FROM sys_user
    WHERE user_name = 'admin'
      AND gmt_delete IS NULL
    ORDER BY id
    LIMIT 1;

    IF v_admin_user_id IS NULL THEN
        RAISE EXCEPTION 'Organization demo data requires the existing development admin account';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM org_employee
        WHERE user_id = v_admin_user_id
          AND NOT (
              source_system_code = v_source_system
              AND source_id = 'DEV-EMP-ZHANG'
          )
    ) THEN
        RAISE EXCEPTION
            'Development admin account is already bound to a non-demo employee; no binding was changed';
    END IF;

    -- Two legal-entity roots make multi-root tree behavior directly testable.
    INSERT INTO org_legal_entity (
        gmt_create,
        gmt_modify,
        state,
        source_system_code,
        source_id,
        source_code,
        code,
        name,
        short_name,
        entity_type,
        parent_id,
        unified_social_credit_code,
        accounting_code,
        status,
        valid_from,
        valid_to,
        source_version,
        source_updated_at,
        last_sync_at,
        source_status,
        source_deleted,
        sync_status,
        last_error,
        local_note,
        local_tags,
        display_order,
        local_handling_status
    )
    VALUES
        (
            v_now, v_now, true, v_source_system, 'DEV-LE-GROUP', 'DEV-LE-GROUP',
            'DEV-LE-GROUP', '验收集团', '验收集团', 'group', NULL, '', 'DEV-GROUP',
            'enabled', CURRENT_DATE - INTERVAL '8 years', NULL, 'demo-v1', v_now,
            v_now, 'enabled', false, 'success', '', '仅用于开发环境验收',
            '["development", "organization-acceptance"]'::jsonb, 10, ''
        ),
        (
            v_now, v_now, true, v_source_system, 'DEV-LE-OVERSEAS',
            'DEV-LE-OVERSEAS', 'DEV-LE-OVERSEAS', '验收海外控股', '海外控股',
            'group', NULL, '', 'DEV-OVERSEAS', 'enabled',
            CURRENT_DATE - INTERVAL '6 years', NULL, 'demo-v1', v_now, v_now,
            'enabled', false, 'success', '', '仅用于开发环境验收',
            '["development", "organization-acceptance"]'::jsonb, 20, ''
        )
    ON CONFLICT (source_system_code, source_id) DO UPDATE
    SET source_code = EXCLUDED.source_code,
        code = EXCLUDED.code,
        name = EXCLUDED.name,
        short_name = EXCLUDED.short_name,
        entity_type = EXCLUDED.entity_type,
        parent_id = EXCLUDED.parent_id,
        unified_social_credit_code = EXCLUDED.unified_social_credit_code,
        accounting_code = EXCLUDED.accounting_code,
        status = EXCLUDED.status,
        valid_from = EXCLUDED.valid_from,
        valid_to = EXCLUDED.valid_to,
        source_version = EXCLUDED.source_version,
        source_updated_at = EXCLUDED.source_updated_at,
        last_sync_at = EXCLUDED.last_sync_at,
        source_status = EXCLUDED.source_status,
        source_deleted = EXCLUDED.source_deleted,
        sync_status = EXCLUDED.sync_status,
        last_error = EXCLUDED.last_error,
        gmt_modify = EXCLUDED.gmt_modify,
        gmt_delete = NULL,
        state = true;

    SELECT id INTO STRICT v_legal_group_id
    FROM org_legal_entity
    WHERE source_system_code = v_source_system AND source_id = 'DEV-LE-GROUP';

    SELECT id INTO STRICT v_legal_overseas_id
    FROM org_legal_entity
    WHERE source_system_code = v_source_system AND source_id = 'DEV-LE-OVERSEAS';

    INSERT INTO org_legal_entity (
        gmt_create,
        gmt_modify,
        state,
        source_system_code,
        source_id,
        source_code,
        code,
        name,
        short_name,
        entity_type,
        parent_id,
        unified_social_credit_code,
        accounting_code,
        status,
        valid_from,
        valid_to,
        source_version,
        source_updated_at,
        last_sync_at,
        source_status,
        source_deleted,
        sync_status,
        last_error,
        local_note,
        local_tags,
        display_order,
        local_handling_status
    )
    VALUES
        (
            v_now, v_now, true, v_source_system, 'DEV-LE-CN', 'DEV-LE-CN',
            'DEV-LE-CN', '验收科技有限公司', '验收科技', 'legal_company',
            v_legal_group_id, '91310000DEMO000001', 'DEV-CN', 'enabled',
            CURRENT_DATE - INTERVAL '7 years', NULL, 'demo-v1', v_now, v_now,
            'enabled', false, 'success', '', '仅用于开发环境验收',
            '["development", "organization-acceptance"]'::jsonb, 11, ''
        ),
        (
            v_now, v_now, true, v_source_system, 'DEV-LE-LOGISTICS',
            'DEV-LE-LOGISTICS', 'DEV-LE-LOGISTICS', '验收物流有限公司', '验收物流',
            'legal_company', v_legal_group_id, '91440300DEMO000002', 'DEV-LOG',
            'enabled', CURRENT_DATE - INTERVAL '5 years', NULL, 'demo-v1', v_now,
            v_now, 'enabled', false, 'success', '', '仅用于开发环境验收',
            '["development", "organization-acceptance"]'::jsonb, 12, ''
        ),
        (
            v_now, v_now, true, v_source_system, 'DEV-LE-SH-BRANCH',
            'DEV-LE-SH-BRANCH', 'DEV-LE-SH-BRANCH', '验收科技上海分公司',
            '上海分公司', 'branch', v_legal_group_id, '91310115DEMO000003',
            'DEV-SH', 'enabled', CURRENT_DATE - INTERVAL '4 years', NULL,
            'demo-v1', v_now, v_now, 'enabled', false, 'success', '',
            '仅用于开发环境验收', '["development", "organization-acceptance"]'::jsonb,
            13, ''
        ),
        (
            v_now, v_now, true, v_source_system, 'DEV-LE-ACCOUNTING',
            'DEV-LE-ACCOUNTING', 'DEV-LE-ACCOUNTING', '验收科技内部核算中心',
            '内部核算中心', 'accounting_unit', v_legal_group_id, '', 'DEV-ACC',
            'enabled', CURRENT_DATE - INTERVAL '3 years', NULL, 'demo-v1', v_now,
            v_now, 'enabled', false, 'success', '', '仅用于开发环境验收',
            '["development", "organization-acceptance"]'::jsonb, 14, ''
        ),
        (
            v_now, v_now, true, v_source_system, 'DEV-LE-SG', 'DEV-LE-SG',
            'DEV-LE-SG', '验收新加坡公司', '新加坡公司', 'legal_company',
            v_legal_overseas_id, '', 'DEV-SG', 'enabled',
            CURRENT_DATE - INTERVAL '2 years', NULL, 'demo-v1', v_now, v_now,
            'enabled', false, 'success', '', '仅用于开发环境验收',
            '["development", "organization-acceptance"]'::jsonb, 21, ''
        )
    ON CONFLICT (source_system_code, source_id) DO UPDATE
    SET source_code = EXCLUDED.source_code,
        code = EXCLUDED.code,
        name = EXCLUDED.name,
        short_name = EXCLUDED.short_name,
        entity_type = EXCLUDED.entity_type,
        parent_id = EXCLUDED.parent_id,
        unified_social_credit_code = EXCLUDED.unified_social_credit_code,
        accounting_code = EXCLUDED.accounting_code,
        status = EXCLUDED.status,
        valid_from = EXCLUDED.valid_from,
        valid_to = EXCLUDED.valid_to,
        source_version = EXCLUDED.source_version,
        source_updated_at = EXCLUDED.source_updated_at,
        last_sync_at = EXCLUDED.last_sync_at,
        source_status = EXCLUDED.source_status,
        source_deleted = EXCLUDED.source_deleted,
        sync_status = EXCLUDED.sync_status,
        last_error = EXCLUDED.last_error,
        gmt_modify = EXCLUDED.gmt_modify,
        gmt_delete = NULL,
        state = true;

    SELECT id INTO STRICT v_legal_cn_id
    FROM org_legal_entity
    WHERE source_system_code = v_source_system AND source_id = 'DEV-LE-CN';

    SELECT id INTO STRICT v_legal_logistics_id
    FROM org_legal_entity
    WHERE source_system_code = v_source_system AND source_id = 'DEV-LE-LOGISTICS';

    -- Correct the two group children that require the China legal entity as parent.
    UPDATE org_legal_entity
    SET parent_id = v_legal_cn_id,
        gmt_modify = v_now
    WHERE source_system_code = v_source_system
      AND source_id IN ('DEV-LE-SH-BRANCH', 'DEV-LE-ACCOUNTING');

    INSERT INTO org_structure (
        gmt_create,
        gmt_modify,
        state,
        code,
        name,
        structure_type,
        source_system_code,
        source_id,
        status,
        is_default,
        valid_from,
        valid_to,
        source_version,
        last_sync_at,
        sync_status
    )
    VALUES
        (
            v_now, v_now, true, 'DEV-STRUCT-ADMIN', '行政管理架构', 'management',
            v_source_system, 'DEV-STRUCT-ADMIN', 'enabled', true,
            CURRENT_DATE - INTERVAL '5 years', NULL, 'demo-v1', v_now, 'success'
        ),
        (
            v_now, v_now, true, 'DEV-STRUCT-REGION', '区域经营架构', 'management',
            v_source_system, 'DEV-STRUCT-REGION', 'enabled', false,
            CURRENT_DATE - INTERVAL '4 years', NULL, 'demo-v1', v_now, 'success'
        )
    ON CONFLICT (code) DO UPDATE
    SET name = EXCLUDED.name,
        structure_type = EXCLUDED.structure_type,
        source_system_code = EXCLUDED.source_system_code,
        source_id = EXCLUDED.source_id,
        status = EXCLUDED.status,
        is_default = EXCLUDED.is_default,
        valid_from = EXCLUDED.valid_from,
        valid_to = EXCLUDED.valid_to,
        source_version = EXCLUDED.source_version,
        last_sync_at = EXCLUDED.last_sync_at,
        sync_status = EXCLUDED.sync_status,
        gmt_modify = EXCLUDED.gmt_modify,
        gmt_delete = NULL,
        state = true;

    SELECT id INTO STRICT v_structure_admin_id
    FROM org_structure
    WHERE code = 'DEV-STRUCT-ADMIN';

    SELECT id INTO STRICT v_structure_region_id
    FROM org_structure
    WHERE code = 'DEV-STRUCT-REGION';

    INSERT INTO org_unit (
        gmt_create,
        gmt_modify,
        state,
        source_system_code,
        source_id,
        source_code,
        code,
        name,
        unit_type,
        primary_legal_entity_id,
        status,
        valid_from,
        valid_to,
        source_version,
        source_updated_at,
        last_sync_at,
        source_status,
        source_deleted,
        sync_status,
        last_error,
        local_note,
        local_tags,
        display_order,
        local_handling_status
    )
    VALUES
        (
            v_now, v_now, true, v_source_system, 'DEV-OU-HQ', 'DEV-OU-HQ',
            'DEV-OU-HQ', '集团总部', 'center', v_legal_group_id, 'enabled',
            CURRENT_DATE - INTERVAL '5 years', NULL, 'demo-v1', v_now, v_now,
            'enabled', false, 'success', '', '仅用于开发环境验收',
            '["development", "organization-acceptance"]'::jsonb, 10, ''
        ),
        (
            v_now, v_now, true, v_source_system, 'DEV-OU-EAST', 'DEV-OU-EAST',
            'DEV-OU-EAST', '华东事业部', 'business_unit', v_legal_cn_id, 'enabled',
            CURRENT_DATE - INTERVAL '4 years', NULL, 'demo-v1', v_now, v_now,
            'enabled', false, 'success', '', '仅用于开发环境验收',
            '["development", "organization-acceptance"]'::jsonb, 20, ''
        ),
        (
            v_now, v_now, true, v_source_system, 'DEV-OU-SOUTH', 'DEV-OU-SOUTH',
            'DEV-OU-SOUTH', '华南事业部', 'business_unit', v_legal_logistics_id,
            'enabled', CURRENT_DATE - INTERVAL '4 years', NULL, 'demo-v1', v_now,
            v_now, 'enabled', false, 'success', '', '仅用于开发环境验收',
            '["development", "organization-acceptance"]'::jsonb, 30, ''
        ),
        (
            v_now, v_now, true, v_source_system, 'DEV-OU-SH-SALES',
            'DEV-OU-SH-SALES', 'DEV-OU-SH-SALES', '上海销售部', 'department',
            v_legal_cn_id, 'enabled', CURRENT_DATE - INTERVAL '3 years', NULL,
            'demo-v1', v_now, v_now, 'enabled', false, 'success', '',
            '仅用于开发环境验收', '["development", "organization-acceptance"]'::jsonb,
            21, ''
        ),
        (
            v_now, v_now, true, v_source_system, 'DEV-OU-SZ-OPS', 'DEV-OU-SZ-OPS',
            'DEV-OU-SZ-OPS', '深圳运营部', 'department', v_legal_logistics_id,
            'enabled', CURRENT_DATE - INTERVAL '3 years', NULL, 'demo-v1', v_now,
            v_now, 'enabled', false, 'success', '', '仅用于开发环境验收',
            '["development", "organization-acceptance"]'::jsonb, 31, ''
        ),
        (
            v_now, v_now, true, v_source_system, 'DEV-OU-SHARED', 'DEV-OU-SHARED',
            'DEV-OU-SHARED', '共享服务中心', 'center', v_legal_cn_id, 'enabled',
            CURRENT_DATE - INTERVAL '3 years', NULL, 'demo-v1', v_now, v_now,
            'enabled', false, 'success', '', '仅用于开发环境验收',
            '["development", "organization-acceptance"]'::jsonb, 40, ''
        ),
        (
            v_now, v_now, true, v_source_system, 'DEV-OU-PROJECT-A',
            'DEV-OU-PROJECT-A', 'DEV-OU-PROJECT-A', '重点项目组', 'project_group',
            v_legal_logistics_id, 'enabled', CURRENT_DATE - INTERVAL '1 year',
            NULL, 'demo-v1', v_now, v_now, 'enabled', false, 'success', '',
            '仅用于开发环境验收', '["development", "organization-acceptance"]'::jsonb,
            50, ''
        ),
        (
            v_now, v_now, true, v_source_system, 'DEV-OU-EAST-REGION',
            'DEV-OU-EAST-REGION', 'DEV-OU-EAST-REGION', '华东区域', 'region',
            v_legal_cn_id, 'enabled', CURRENT_DATE - INTERVAL '4 years', NULL,
            'demo-v1', v_now, v_now, 'enabled', false, 'success', '',
            '仅用于开发环境验收', '["development", "organization-acceptance"]'::jsonb,
            60, ''
        ),
        (
            v_now, v_now, true, v_source_system, 'DEV-OU-SOUTH-REGION',
            'DEV-OU-SOUTH-REGION', 'DEV-OU-SOUTH-REGION', '华南区域', 'region',
            v_legal_logistics_id, 'enabled', CURRENT_DATE - INTERVAL '4 years',
            NULL, 'demo-v1', v_now, v_now, 'enabled', false, 'success', '',
            '仅用于开发环境验收', '["development", "organization-acceptance"]'::jsonb,
            70, ''
        )
    ON CONFLICT (source_system_code, source_id) DO UPDATE
    SET source_code = EXCLUDED.source_code,
        code = EXCLUDED.code,
        name = EXCLUDED.name,
        unit_type = EXCLUDED.unit_type,
        primary_legal_entity_id = EXCLUDED.primary_legal_entity_id,
        status = EXCLUDED.status,
        valid_from = EXCLUDED.valid_from,
        valid_to = EXCLUDED.valid_to,
        source_version = EXCLUDED.source_version,
        source_updated_at = EXCLUDED.source_updated_at,
        last_sync_at = EXCLUDED.last_sync_at,
        source_status = EXCLUDED.source_status,
        source_deleted = EXCLUDED.source_deleted,
        sync_status = EXCLUDED.sync_status,
        last_error = EXCLUDED.last_error,
        gmt_modify = EXCLUDED.gmt_modify,
        gmt_delete = NULL,
        state = true;

    SELECT id INTO STRICT v_unit_hq_id FROM org_unit
    WHERE source_system_code = v_source_system AND source_id = 'DEV-OU-HQ';
    SELECT id INTO STRICT v_unit_east_id FROM org_unit
    WHERE source_system_code = v_source_system AND source_id = 'DEV-OU-EAST';
    SELECT id INTO STRICT v_unit_south_id FROM org_unit
    WHERE source_system_code = v_source_system AND source_id = 'DEV-OU-SOUTH';
    SELECT id INTO STRICT v_unit_sales_id FROM org_unit
    WHERE source_system_code = v_source_system AND source_id = 'DEV-OU-SH-SALES';
    SELECT id INTO STRICT v_unit_ops_id FROM org_unit
    WHERE source_system_code = v_source_system AND source_id = 'DEV-OU-SZ-OPS';
    SELECT id INTO STRICT v_unit_shared_id FROM org_unit
    WHERE source_system_code = v_source_system AND source_id = 'DEV-OU-SHARED';
    SELECT id INTO STRICT v_unit_project_id FROM org_unit
    WHERE source_system_code = v_source_system AND source_id = 'DEV-OU-PROJECT-A';
    SELECT id INTO STRICT v_unit_east_region_id FROM org_unit
    WHERE source_system_code = v_source_system AND source_id = 'DEV-OU-EAST-REGION';
    SELECT id INTO STRICT v_unit_south_region_id FROM org_unit
    WHERE source_system_code = v_source_system AND source_id = 'DEV-OU-SOUTH-REGION';

    -- Administrative structure.
    INSERT INTO org_structure_node (
        gmt_create, gmt_modify, state, structure_id, org_unit_id, parent_node_id,
        source_system_code, source_id, source_parent_id, path, level, sort,
        valid_from, valid_to, status, source_deleted, sync_status
    )
    VALUES (
        v_now, v_now, true, v_structure_admin_id, v_unit_hq_id, NULL,
        v_source_system, 'DEV-NODE-ADMIN-HQ', '', '/pending/DEV-NODE-ADMIN-HQ',
        1, 10, CURRENT_DATE - INTERVAL '5 years', NULL, 'enabled', false, 'success'
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
        valid_to = EXCLUDED.valid_to,
        status = EXCLUDED.status,
        source_deleted = EXCLUDED.source_deleted,
        sync_status = EXCLUDED.sync_status,
        gmt_modify = EXCLUDED.gmt_modify,
        gmt_delete = NULL,
        state = true
    RETURNING id INTO v_admin_hq_node_id;

    UPDATE org_structure_node
    SET path = format('/%s', v_admin_hq_node_id)
    WHERE id = v_admin_hq_node_id;

    INSERT INTO org_structure_node (
        gmt_create, gmt_modify, state, structure_id, org_unit_id, parent_node_id,
        source_system_code, source_id, source_parent_id, path, level, sort,
        valid_from, valid_to, status, source_deleted, sync_status
    )
    VALUES
        (
            v_now, v_now, true, v_structure_admin_id, v_unit_east_id,
            v_admin_hq_node_id, v_source_system, 'DEV-NODE-ADMIN-EAST',
            'DEV-NODE-ADMIN-HQ', '/pending/DEV-NODE-ADMIN-EAST', 2, 10,
            CURRENT_DATE - INTERVAL '4 years', NULL, 'enabled', false, 'success'
        ),
        (
            v_now, v_now, true, v_structure_admin_id, v_unit_south_id,
            v_admin_hq_node_id, v_source_system, 'DEV-NODE-ADMIN-SOUTH',
            'DEV-NODE-ADMIN-HQ', '/pending/DEV-NODE-ADMIN-SOUTH', 2, 20,
            CURRENT_DATE - INTERVAL '4 years', NULL, 'enabled', false, 'success'
        ),
        (
            v_now, v_now, true, v_structure_admin_id, v_unit_shared_id,
            v_admin_hq_node_id, v_source_system, 'DEV-NODE-ADMIN-SHARED',
            'DEV-NODE-ADMIN-HQ', '/pending/DEV-NODE-ADMIN-SHARED', 2, 30,
            CURRENT_DATE - INTERVAL '3 years', NULL, 'enabled', false, 'success'
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
        valid_to = EXCLUDED.valid_to,
        status = EXCLUDED.status,
        source_deleted = EXCLUDED.source_deleted,
        sync_status = EXCLUDED.sync_status,
        gmt_modify = EXCLUDED.gmt_modify,
        gmt_delete = NULL,
        state = true;

    SELECT id INTO STRICT v_admin_east_node_id FROM org_structure_node
    WHERE source_system_code = v_source_system AND source_id = 'DEV-NODE-ADMIN-EAST';
    SELECT id INTO STRICT v_admin_south_node_id FROM org_structure_node
    WHERE source_system_code = v_source_system AND source_id = 'DEV-NODE-ADMIN-SOUTH';

    UPDATE org_structure_node
    SET path = format('/%s/%s', v_admin_hq_node_id, id)
    WHERE source_system_code = v_source_system
      AND source_id IN (
          'DEV-NODE-ADMIN-EAST',
          'DEV-NODE-ADMIN-SOUTH',
          'DEV-NODE-ADMIN-SHARED'
      );

    INSERT INTO org_structure_node (
        gmt_create, gmt_modify, state, structure_id, org_unit_id, parent_node_id,
        source_system_code, source_id, source_parent_id, path, level, sort,
        valid_from, valid_to, status, source_deleted, sync_status
    )
    VALUES
        (
            v_now, v_now, true, v_structure_admin_id, v_unit_sales_id,
            v_admin_east_node_id, v_source_system, 'DEV-NODE-ADMIN-SALES',
            'DEV-NODE-ADMIN-EAST', '/pending/DEV-NODE-ADMIN-SALES', 3, 10,
            CURRENT_DATE - INTERVAL '3 years', NULL, 'enabled', false, 'success'
        ),
        (
            v_now, v_now, true, v_structure_admin_id, v_unit_ops_id,
            v_admin_south_node_id, v_source_system, 'DEV-NODE-ADMIN-OPS',
            'DEV-NODE-ADMIN-SOUTH', '/pending/DEV-NODE-ADMIN-OPS', 3, 10,
            CURRENT_DATE - INTERVAL '3 years', NULL, 'enabled', false, 'success'
        ),
        (
            v_now, v_now, true, v_structure_admin_id, v_unit_project_id,
            v_admin_south_node_id, v_source_system, 'DEV-NODE-ADMIN-PROJECT',
            'DEV-NODE-ADMIN-SOUTH', '/pending/DEV-NODE-ADMIN-PROJECT', 3, 20,
            CURRENT_DATE - INTERVAL '1 year', NULL, 'enabled', false, 'success'
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
        valid_to = EXCLUDED.valid_to,
        status = EXCLUDED.status,
        source_deleted = EXCLUDED.source_deleted,
        sync_status = EXCLUDED.sync_status,
        gmt_modify = EXCLUDED.gmt_modify,
        gmt_delete = NULL,
        state = true;

    UPDATE org_structure_node
    SET path = CASE
        WHEN source_id = 'DEV-NODE-ADMIN-SALES'
            THEN format('/%s/%s/%s', v_admin_hq_node_id, v_admin_east_node_id, id)
        ELSE format('/%s/%s/%s', v_admin_hq_node_id, v_admin_south_node_id, id)
    END
    WHERE source_system_code = v_source_system
      AND source_id IN (
          'DEV-NODE-ADMIN-SALES',
          'DEV-NODE-ADMIN-OPS',
          'DEV-NODE-ADMIN-PROJECT'
      );

    -- Regional structure reuses business org units with distinct runtime nodes.
    INSERT INTO org_structure_node (
        gmt_create, gmt_modify, state, structure_id, org_unit_id, parent_node_id,
        source_system_code, source_id, source_parent_id, path, level, sort,
        valid_from, valid_to, status, source_deleted, sync_status
    )
    VALUES
        (
            v_now, v_now, true, v_structure_region_id, v_unit_east_region_id, NULL,
            v_source_system, 'DEV-NODE-REGION-EAST', '',
            '/pending/DEV-NODE-REGION-EAST', 1, 10,
            CURRENT_DATE - INTERVAL '4 years', NULL, 'enabled', false, 'success'
        ),
        (
            v_now, v_now, true, v_structure_region_id, v_unit_south_region_id, NULL,
            v_source_system, 'DEV-NODE-REGION-SOUTH', '',
            '/pending/DEV-NODE-REGION-SOUTH', 1, 20,
            CURRENT_DATE - INTERVAL '4 years', NULL, 'enabled', false, 'success'
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
        valid_to = EXCLUDED.valid_to,
        status = EXCLUDED.status,
        source_deleted = EXCLUDED.source_deleted,
        sync_status = EXCLUDED.sync_status,
        gmt_modify = EXCLUDED.gmt_modify,
        gmt_delete = NULL,
        state = true;

    SELECT id INTO STRICT v_region_east_node_id FROM org_structure_node
    WHERE source_system_code = v_source_system AND source_id = 'DEV-NODE-REGION-EAST';
    SELECT id INTO STRICT v_region_south_node_id FROM org_structure_node
    WHERE source_system_code = v_source_system AND source_id = 'DEV-NODE-REGION-SOUTH';

    UPDATE org_structure_node
    SET path = format('/%s', id)
    WHERE source_system_code = v_source_system
      AND source_id IN ('DEV-NODE-REGION-EAST', 'DEV-NODE-REGION-SOUTH');

    INSERT INTO org_structure_node (
        gmt_create, gmt_modify, state, structure_id, org_unit_id, parent_node_id,
        source_system_code, source_id, source_parent_id, path, level, sort,
        valid_from, valid_to, status, source_deleted, sync_status
    )
    VALUES
        (
            v_now, v_now, true, v_structure_region_id, v_unit_sales_id,
            v_region_east_node_id, v_source_system, 'DEV-NODE-REGION-SALES',
            'DEV-NODE-REGION-EAST', '/pending/DEV-NODE-REGION-SALES', 2, 10,
            CURRENT_DATE - INTERVAL '3 years', NULL, 'enabled', false, 'success'
        ),
        (
            v_now, v_now, true, v_structure_region_id, v_unit_shared_id,
            v_region_east_node_id, v_source_system, 'DEV-NODE-REGION-SHARED',
            'DEV-NODE-REGION-EAST', '/pending/DEV-NODE-REGION-SHARED', 2, 20,
            CURRENT_DATE - INTERVAL '3 years', NULL, 'enabled', false, 'success'
        ),
        (
            v_now, v_now, true, v_structure_region_id, v_unit_ops_id,
            v_region_south_node_id, v_source_system, 'DEV-NODE-REGION-OPS',
            'DEV-NODE-REGION-SOUTH', '/pending/DEV-NODE-REGION-OPS', 2, 10,
            CURRENT_DATE - INTERVAL '3 years', NULL, 'enabled', false, 'success'
        ),
        (
            v_now, v_now, true, v_structure_region_id, v_unit_project_id,
            v_region_south_node_id, v_source_system, 'DEV-NODE-REGION-PROJECT',
            'DEV-NODE-REGION-SOUTH', '/pending/DEV-NODE-REGION-PROJECT', 2, 20,
            CURRENT_DATE - INTERVAL '1 year', NULL, 'enabled', false, 'success'
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
        valid_to = EXCLUDED.valid_to,
        status = EXCLUDED.status,
        source_deleted = EXCLUDED.source_deleted,
        sync_status = EXCLUDED.sync_status,
        gmt_modify = EXCLUDED.gmt_modify,
        gmt_delete = NULL,
        state = true;

    UPDATE org_structure_node
    SET path = CASE
        WHEN source_id IN ('DEV-NODE-REGION-SALES', 'DEV-NODE-REGION-SHARED')
            THEN format('/%s/%s', v_region_east_node_id, id)
        ELSE format('/%s/%s', v_region_south_node_id, id)
    END
    WHERE source_system_code = v_source_system
      AND source_id IN (
          'DEV-NODE-REGION-SALES',
          'DEV-NODE-REGION-SHARED',
          'DEV-NODE-REGION-OPS',
          'DEV-NODE-REGION-PROJECT'
      );

    INSERT INTO org_position (
        gmt_create,
        gmt_modify,
        state,
        source_system_code,
        source_id,
        source_code,
        code,
        name,
        org_unit_id,
        position_type,
        job_level,
        is_manager_position,
        status,
        valid_from,
        valid_to,
        source_version,
        last_sync_at,
        source_deleted,
        sync_status,
        local_note
    )
    VALUES
        (
            v_now, v_now, true, v_source_system, 'DEV-POS-HQ-DIRECTOR',
            'DEV-POS-HQ-DIRECTOR', 'DEV-POS-HQ-DIRECTOR', '总部负责人',
            v_unit_hq_id, 'management', 'M3', true, 'enabled',
            CURRENT_DATE - INTERVAL '5 years', NULL, 'demo-v1', v_now, false,
            'success', '仅用于开发环境验收'
        ),
        (
            v_now, v_now, true, v_source_system, 'DEV-POS-EAST-MANAGER',
            'DEV-POS-EAST-MANAGER', 'DEV-POS-EAST-MANAGER', '华东事业部负责人',
            v_unit_east_id, 'management', 'M2', true, 'enabled',
            CURRENT_DATE - INTERVAL '4 years', NULL, 'demo-v1', v_now, false,
            'success', '仅用于开发环境验收'
        ),
        (
            v_now, v_now, true, v_source_system, 'DEV-POS-SALES-MANAGER',
            'DEV-POS-SALES-MANAGER', 'DEV-POS-SALES-MANAGER', '销售经理',
            v_unit_sales_id, 'management', 'M1', true, 'enabled',
            CURRENT_DATE - INTERVAL '3 years', NULL, 'demo-v1', v_now, false,
            'success', '仅用于开发环境验收'
        ),
        (
            v_now, v_now, true, v_source_system, 'DEV-POS-SALES-SPECIALIST',
            'DEV-POS-SALES-SPECIALIST', 'DEV-POS-SALES-SPECIALIST', '销售专员',
            v_unit_sales_id, 'professional', 'P2', false, 'enabled',
            CURRENT_DATE - INTERVAL '3 years', NULL, 'demo-v1', v_now, false,
            'success', '仅用于开发环境验收'
        ),
        (
            v_now, v_now, true, v_source_system, 'DEV-POS-SHARED-FINANCE',
            'DEV-POS-SHARED-FINANCE', 'DEV-POS-SHARED-FINANCE', '共享财务专员',
            v_unit_shared_id, 'professional', 'P2', false, 'enabled',
            CURRENT_DATE - INTERVAL '3 years', NULL, 'demo-v1', v_now, false,
            'success', '仅用于开发环境验收'
        ),
        (
            v_now, v_now, true, v_source_system, 'DEV-POS-OPS',
            'DEV-POS-OPS', 'DEV-POS-OPS', '运营专员', v_unit_ops_id, 'operation',
            'P2', false, 'enabled', CURRENT_DATE - INTERVAL '3 years', NULL,
            'demo-v1', v_now, false, 'success', '仅用于开发环境验收'
        ),
        (
            v_now, v_now, true, v_source_system, 'DEV-POS-PROJECT',
            'DEV-POS-PROJECT', 'DEV-POS-PROJECT', '项目协调员', v_unit_project_id,
            'professional', 'P3', false, 'enabled',
            CURRENT_DATE - INTERVAL '1 year', NULL, 'demo-v1', v_now, false,
            'success', '仅用于开发环境验收'
        ),
        (
            v_now, v_now, true, v_source_system, 'DEV-POS-LEGACY-SALES',
            'DEV-POS-LEGACY-SALES', 'DEV-POS-LEGACY-SALES', '历史销售岗位',
            v_unit_sales_id, 'professional', 'P1', false, 'disabled',
            CURRENT_DATE - INTERVAL '6 years', CURRENT_DATE - INTERVAL '1 year',
            'demo-v1', v_now, false, 'success', '仅用于历史回显验收'
        )
    ON CONFLICT (source_system_code, source_id) DO UPDATE
    SET source_code = EXCLUDED.source_code,
        code = EXCLUDED.code,
        name = EXCLUDED.name,
        org_unit_id = EXCLUDED.org_unit_id,
        position_type = EXCLUDED.position_type,
        job_level = EXCLUDED.job_level,
        is_manager_position = EXCLUDED.is_manager_position,
        status = EXCLUDED.status,
        valid_from = EXCLUDED.valid_from,
        valid_to = EXCLUDED.valid_to,
        source_version = EXCLUDED.source_version,
        last_sync_at = EXCLUDED.last_sync_at,
        source_deleted = EXCLUDED.source_deleted,
        sync_status = EXCLUDED.sync_status,
        gmt_modify = EXCLUDED.gmt_modify,
        gmt_delete = NULL,
        state = true;

    SELECT id INTO STRICT v_position_sales_manager_id FROM org_position
    WHERE source_system_code = v_source_system AND source_id = 'DEV-POS-SALES-MANAGER';
    SELECT id INTO STRICT v_position_sales_specialist_id FROM org_position
    WHERE source_system_code = v_source_system AND source_id = 'DEV-POS-SALES-SPECIALIST';
    SELECT id INTO STRICT v_position_shared_finance_id FROM org_position
    WHERE source_system_code = v_source_system AND source_id = 'DEV-POS-SHARED-FINANCE';
    SELECT id INTO STRICT v_position_ops_id FROM org_position
    WHERE source_system_code = v_source_system AND source_id = 'DEV-POS-OPS';
    SELECT id INTO STRICT v_position_project_id FROM org_position
    WHERE source_system_code = v_source_system AND source_id = 'DEV-POS-PROJECT';
    SELECT id INTO STRICT v_position_legacy_id FROM org_position
    WHERE source_system_code = v_source_system AND source_id = 'DEV-POS-LEGACY-SALES';

    INSERT INTO org_employee (
        gmt_create,
        gmt_modify,
        state,
        source_system_code,
        source_id,
        source_code,
        employee_no,
        name,
        mobile,
        email,
        employment_status,
        primary_legal_entity_id,
        valid_from,
        valid_to,
        source_version,
        source_updated_at,
        last_sync_at,
        source_deleted,
        sync_status,
        local_note,
        local_tags
    )
    VALUES
        (
            v_now, v_now, true, v_source_system, 'DEV-EMP-ZHANG', 'DEV-EMP-ZHANG',
            'DEV-E0001', '张伟', '13800000001', 'zhang.wei@example.invalid',
            'active', v_legal_cn_id, CURRENT_DATE - INTERVAL '5 years', NULL,
            'demo-v1', v_now, v_now, false, 'success', '仅用于开发环境验收',
            '["development", "organization-acceptance"]'::jsonb
        ),
        (
            v_now, v_now, true, v_source_system, 'DEV-EMP-LI', 'DEV-EMP-LI',
            'DEV-E0002', '李娜', '13800000002', 'li.na@example.invalid', 'active',
            v_legal_cn_id, CURRENT_DATE - INTERVAL '3 years', NULL, 'demo-v1',
            v_now, v_now, false, 'success', '仅用于开发环境验收',
            '["development", "organization-acceptance"]'::jsonb
        ),
        (
            v_now, v_now, true, v_source_system, 'DEV-EMP-WANG', 'DEV-EMP-WANG',
            'DEV-E0003', '王强', '13800000003', 'wang.qiang@example.invalid',
            'active', v_legal_logistics_id, CURRENT_DATE - INTERVAL '2 years',
            NULL, 'demo-v1', v_now, v_now, false, 'success', '仅用于开发环境验收',
            '["development", "organization-acceptance"]'::jsonb
        ),
        (
            v_now, v_now, true, v_source_system, 'DEV-EMP-HISTORY',
            'DEV-EMP-HISTORY', 'DEV-E0098', '陈历史', '13800000098',
            'history.employee@example.invalid', 'resigned', v_legal_cn_id,
            CURRENT_DATE - INTERVAL '7 years', CURRENT_DATE - INTERVAL '1 year',
            'demo-v1', v_now, v_now, false, 'success', '仅用于历史回显验收',
            '["development", "organization-acceptance"]'::jsonb
        ),
        (
            v_now, v_now, true, v_source_system, 'DEV-EMP-FUTURE',
            'DEV-EMP-FUTURE', 'DEV-E0099', '赵未来', '13800000099',
            'future.employee@example.invalid', 'probation', v_legal_logistics_id,
            CURRENT_DATE + INTERVAL '30 days', NULL, 'demo-v1', v_now, v_now,
            false, 'success', '仅用于未来有效期验收',
            '["development", "organization-acceptance"]'::jsonb
        )
    ON CONFLICT (source_system_code, source_id) DO UPDATE
    SET source_code = EXCLUDED.source_code,
        employee_no = EXCLUDED.employee_no,
        name = EXCLUDED.name,
        mobile = EXCLUDED.mobile,
        email = EXCLUDED.email,
        employment_status = EXCLUDED.employment_status,
        primary_legal_entity_id = EXCLUDED.primary_legal_entity_id,
        valid_from = EXCLUDED.valid_from,
        valid_to = EXCLUDED.valid_to,
        source_version = EXCLUDED.source_version,
        source_updated_at = EXCLUDED.source_updated_at,
        last_sync_at = EXCLUDED.last_sync_at,
        source_deleted = EXCLUDED.source_deleted,
        sync_status = EXCLUDED.sync_status,
        gmt_modify = EXCLUDED.gmt_modify,
        gmt_delete = NULL,
        state = true;

    SELECT id INTO STRICT v_employee_zhang_id FROM org_employee
    WHERE source_system_code = v_source_system AND source_id = 'DEV-EMP-ZHANG';
    SELECT id INTO STRICT v_employee_li_id FROM org_employee
    WHERE source_system_code = v_source_system AND source_id = 'DEV-EMP-LI';
    SELECT id INTO STRICT v_employee_wang_id FROM org_employee
    WHERE source_system_code = v_source_system AND source_id = 'DEV-EMP-WANG';
    SELECT id INTO STRICT v_employee_history_id FROM org_employee
    WHERE source_system_code = v_source_system AND source_id = 'DEV-EMP-HISTORY';
    SELECT id INTO STRICT v_employee_future_id FROM org_employee
    WHERE source_system_code = v_source_system AND source_id = 'DEV-EMP-FUTURE';

    -- user_id is a platform extension field and is changed explicitly, outside source-field upsert.
    UPDATE org_employee
    SET user_id = v_admin_user_id,
        gmt_modify = v_now
    WHERE id = v_employee_zhang_id;

    INSERT INTO org_assignment (
        gmt_create,
        gmt_modify,
        state,
        source_system_code,
        source_id,
        employee_id,
        legal_entity_id,
        org_unit_id,
        position_id,
        assignment_type,
        is_primary,
        is_manager,
        valid_from,
        valid_to,
        status,
        source_version,
        source_deleted,
        sync_status
    )
    VALUES
        (
            v_now, v_now, true, v_source_system, 'DEV-ASG-ZHANG-HISTORY',
            v_employee_zhang_id, v_legal_cn_id, v_unit_east_id,
            v_position_legacy_id, 'primary', false, false,
            CURRENT_DATE - INTERVAL '5 years', CURRENT_DATE - INTERVAL '1 year',
            'enabled', 'demo-v1', false, 'success'
        ),
        (
            v_now, v_now, true, v_source_system, 'DEV-ASG-ZHANG-CURRENT-PRIMARY',
            v_employee_zhang_id, v_legal_cn_id, v_unit_sales_id,
            v_position_sales_manager_id, 'primary', true, true,
            CURRENT_DATE - INTERVAL '1 year', NULL, 'enabled', 'demo-v1', false,
            'success'
        ),
        (
            v_now, v_now, true, v_source_system, 'DEV-ASG-ZHANG-CURRENT-PART',
            v_employee_zhang_id, v_legal_cn_id, v_unit_shared_id,
            v_position_shared_finance_id, 'part_time', false, false,
            CURRENT_DATE - INTERVAL '90 days', NULL, 'enabled', 'demo-v1', false,
            'success'
        ),
        (
            v_now, v_now, true, v_source_system, 'DEV-ASG-ZHANG-FUTURE',
            v_employee_zhang_id, v_legal_logistics_id, v_unit_project_id,
            v_position_project_id, 'project', false, false,
            CURRENT_DATE + INTERVAL '30 days', NULL, 'enabled', 'demo-v1', false,
            'success'
        ),
        (
            v_now, v_now, true, v_source_system, 'DEV-ASG-LI-CURRENT',
            v_employee_li_id, v_legal_cn_id, v_unit_sales_id,
            v_position_sales_specialist_id, 'primary', true, false,
            CURRENT_DATE - INTERVAL '2 years', NULL, 'enabled', 'demo-v1', false,
            'success'
        ),
        (
            v_now, v_now, true, v_source_system, 'DEV-ASG-WANG-CURRENT',
            v_employee_wang_id, v_legal_logistics_id, v_unit_ops_id,
            v_position_ops_id, 'primary', true, false,
            CURRENT_DATE - INTERVAL '1 year', NULL, 'enabled', 'demo-v1', false,
            'success'
        ),
        (
            v_now, v_now, true, v_source_system, 'DEV-ASG-HISTORY',
            v_employee_history_id, v_legal_cn_id, v_unit_sales_id,
            v_position_legacy_id, 'primary', false, false,
            CURRENT_DATE - INTERVAL '6 years', CURRENT_DATE - INTERVAL '1 year',
            'enabled', 'demo-v1', false, 'success'
        ),
        (
            v_now, v_now, true, v_source_system, 'DEV-ASG-FUTURE',
            v_employee_future_id, v_legal_logistics_id, v_unit_project_id,
            v_position_project_id, 'primary', true, false,
            CURRENT_DATE + INTERVAL '30 days', NULL, 'enabled', 'demo-v1', false,
            'success'
        )
    ON CONFLICT (source_system_code, source_id) DO UPDATE
    SET employee_id = EXCLUDED.employee_id,
        legal_entity_id = EXCLUDED.legal_entity_id,
        org_unit_id = EXCLUDED.org_unit_id,
        position_id = EXCLUDED.position_id,
        assignment_type = EXCLUDED.assignment_type,
        is_primary = EXCLUDED.is_primary,
        is_manager = EXCLUDED.is_manager,
        valid_from = EXCLUDED.valid_from,
        valid_to = EXCLUDED.valid_to,
        status = EXCLUDED.status,
        source_version = EXCLUDED.source_version,
        source_deleted = EXCLUDED.source_deleted,
        sync_status = EXCLUDED.sync_status,
        gmt_modify = EXCLUDED.gmt_modify,
        gmt_delete = NULL,
        state = true;

    -- Minimal business synchronization outcomes; no technical HTTP payload is duplicated here.
    INSERT INTO org_sync_batch (
        gmt_create,
        gmt_modify,
        state,
        batch_no,
        execution_id,
        sync_type,
        object_scope,
        started_at,
        completed_at,
        total_count,
        success_count,
        failed_count,
        skipped_count,
        status,
        error_summary
    )
    VALUES
        (
            v_now, v_now, true, 'DEV-ORG-BATCH-SUCCESS', NULL, 'full', 'all',
            v_now - INTERVAL '15 minutes', v_now - INTERVAL '10 minutes',
            3, 2, 0, 1, 'success', ''
        ),
        (
            v_now, v_now, true, 'DEV-ORG-BATCH-FAILED', NULL, 'incremental',
            'assignment', v_now - INTERVAL '5 minutes', v_now - INTERVAL '4 minutes',
            1, 0, 1, 0, 'failed', '开发验收：模拟任职依赖错误'
        )
    ON CONFLICT (batch_no) DO UPDATE
    SET execution_id = EXCLUDED.execution_id,
        sync_type = EXCLUDED.sync_type,
        object_scope = EXCLUDED.object_scope,
        started_at = EXCLUDED.started_at,
        completed_at = EXCLUDED.completed_at,
        total_count = EXCLUDED.total_count,
        success_count = EXCLUDED.success_count,
        failed_count = EXCLUDED.failed_count,
        skipped_count = EXCLUDED.skipped_count,
        status = EXCLUDED.status,
        error_summary = EXCLUDED.error_summary,
        gmt_modify = EXCLUDED.gmt_modify,
        gmt_delete = NULL,
        state = true;

    SELECT id INTO STRICT v_batch_success_id
    FROM org_sync_batch WHERE batch_no = 'DEV-ORG-BATCH-SUCCESS';
    SELECT id INTO STRICT v_batch_failed_id
    FROM org_sync_batch WHERE batch_no = 'DEV-ORG-BATCH-FAILED';

    SELECT id INTO v_sync_record_id
    FROM org_sync_record
    WHERE batch_id = v_batch_success_id
      AND object_type = 'legal_entity'
      AND source_id = 'DEV-LE-CN'
      AND action = 'update'
    ORDER BY id
    LIMIT 1;

    IF v_sync_record_id IS NULL THEN
        INSERT INTO org_sync_record (
            gmt_create, gmt_modify, state, batch_id, execution_id, object_type,
            source_id, source_code, local_id, action, status, error_code,
            error_message, dependency_type, dependency_key, retry_count,
            last_retry_at, local_handling_status
        )
        VALUES (
            v_now, v_now, true, v_batch_success_id, NULL, 'legal_entity',
            'DEV-LE-CN', 'DEV-LE-CN', v_legal_cn_id, 'update', 'success', '',
            '', '', '', 0, NULL, ''
        );
    ELSE
        UPDATE org_sync_record
        SET local_id = v_legal_cn_id,
            status = 'success',
            error_code = '',
            error_message = '',
            dependency_type = '',
            dependency_key = '',
            retry_count = 0,
            last_retry_at = NULL,
            local_handling_status = '',
            gmt_modify = v_now,
            gmt_delete = NULL,
            state = true
        WHERE id = v_sync_record_id;
    END IF;

    v_sync_record_id := NULL;
    SELECT id INTO v_sync_record_id
    FROM org_sync_record
    WHERE batch_id = v_batch_success_id
      AND object_type = 'org_unit'
      AND source_id = 'DEV-OU-SHARED'
      AND action = 'no_change'
    ORDER BY id
    LIMIT 1;

    IF v_sync_record_id IS NULL THEN
        INSERT INTO org_sync_record (
            gmt_create, gmt_modify, state, batch_id, execution_id, object_type,
            source_id, source_code, local_id, action, status, error_code,
            error_message, dependency_type, dependency_key, retry_count,
            last_retry_at, local_handling_status
        )
        VALUES (
            v_now, v_now, true, v_batch_success_id, NULL, 'org_unit',
            'DEV-OU-SHARED', 'DEV-OU-SHARED', v_unit_shared_id, 'no_change',
            'success', '', '', '', '', 0, NULL, ''
        );
    ELSE
        UPDATE org_sync_record
        SET local_id = v_unit_shared_id,
            status = 'success',
            error_code = '',
            error_message = '',
            dependency_type = '',
            dependency_key = '',
            retry_count = 0,
            last_retry_at = NULL,
            local_handling_status = '',
            gmt_modify = v_now,
            gmt_delete = NULL,
            state = true
        WHERE id = v_sync_record_id;
    END IF;

    v_sync_record_id := NULL;
    SELECT id INTO v_sync_record_id
    FROM org_sync_record
    WHERE batch_id = v_batch_failed_id
      AND object_type = 'assignment'
      AND source_id = 'DEV-ASG-DEPENDENCY-ERROR'
      AND action = 'insert'
    ORDER BY id
    LIMIT 1;

    IF v_sync_record_id IS NULL THEN
        INSERT INTO org_sync_record (
            gmt_create, gmt_modify, state, batch_id, execution_id, object_type,
            source_id, source_code, local_id, action, status, error_code,
            error_message, dependency_type, dependency_key, retry_count,
            last_retry_at, local_handling_status
        )
        VALUES (
            v_now, v_now, true, v_batch_failed_id, NULL, 'assignment',
            'DEV-ASG-DEPENDENCY-ERROR', '', NULL, 'insert', 'failed',
            'org_position_missing', '开发验收：引用的岗位尚未到达', 'position',
            'DEV-POS-MISSING', 1, v_now - INTERVAL '2 minutes', 'pending'
        );
    ELSE
        UPDATE org_sync_record
        SET local_id = NULL,
            status = 'failed',
            error_code = 'org_position_missing',
            error_message = '开发验收：引用的岗位尚未到达',
            dependency_type = 'position',
            dependency_key = 'DEV-POS-MISSING',
            retry_count = 1,
            last_retry_at = v_now - INTERVAL '2 minutes',
            local_handling_status = 'pending',
            gmt_modify = v_now,
            gmt_delete = NULL,
            state = true
        WHERE id = v_sync_record_id;
    END IF;

    -- Self-checks make repeat execution and relationship regressions visible.
    IF (SELECT count(*) FROM org_legal_entity
        WHERE source_system_code = v_source_system AND gmt_delete IS NULL) <> 7 THEN
        RAISE EXCEPTION 'Expected 7 Organization demo legal entities';
    END IF;

    IF (SELECT count(*) FROM org_structure
        WHERE source_system_code = v_source_system AND gmt_delete IS NULL) <> 2 THEN
        RAISE EXCEPTION 'Expected 2 Organization demo structures';
    END IF;

    IF (SELECT count(*) FROM org_unit
        WHERE source_system_code = v_source_system AND gmt_delete IS NULL) <> 9 THEN
        RAISE EXCEPTION 'Expected 9 Organization demo units';
    END IF;

    IF (SELECT count(*) FROM org_structure_node
        WHERE source_system_code = v_source_system AND gmt_delete IS NULL) <> 13 THEN
        RAISE EXCEPTION 'Expected 13 Organization demo structure nodes';
    END IF;

    IF (SELECT count(*) FROM org_position
        WHERE source_system_code = v_source_system AND gmt_delete IS NULL) <> 8 THEN
        RAISE EXCEPTION 'Expected 8 Organization demo positions';
    END IF;

    IF (SELECT count(*) FROM org_employee
        WHERE source_system_code = v_source_system AND gmt_delete IS NULL) <> 5 THEN
        RAISE EXCEPTION 'Expected 5 Organization demo employees';
    END IF;

    IF (SELECT count(*) FROM org_assignment
        WHERE source_system_code = v_source_system AND gmt_delete IS NULL) <> 8 THEN
        RAISE EXCEPTION 'Expected 8 Organization demo assignments';
    END IF;

    IF (
        SELECT count(DISTINCT structure_id)
        FROM org_structure_node
        WHERE source_system_code = v_source_system
          AND org_unit_id = v_unit_shared_id
          AND gmt_delete IS NULL
    ) <> 2 THEN
        RAISE EXCEPTION 'Shared service org unit must occur in both demo structures';
    END IF;

    IF (
        SELECT count(*)
        FROM org_assignment
        WHERE employee_id = v_employee_zhang_id
          AND status = 'enabled'
          AND source_deleted = false
          AND valid_from <= CURRENT_DATE
          AND (valid_to IS NULL OR valid_to >= CURRENT_DATE)
          AND gmt_delete IS NULL
    ) <> 2 THEN
        RAISE EXCEPTION 'Zhang Wei must have two current assignments';
    END IF;

    IF (
        SELECT count(*)
        FROM org_assignment
        WHERE employee_id = v_employee_zhang_id
          AND valid_to < CURRENT_DATE
          AND gmt_delete IS NULL
    ) <> 1 THEN
        RAISE EXCEPTION 'Zhang Wei must have one historical assignment';
    END IF;

    IF (
        SELECT count(*)
        FROM org_assignment
        WHERE employee_id = v_employee_zhang_id
          AND valid_from > CURRENT_DATE
          AND gmt_delete IS NULL
    ) <> 1 THEN
        RAISE EXCEPTION 'Zhang Wei must have one future assignment';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM org_employee
        WHERE id = v_employee_zhang_id
          AND user_id = v_admin_user_id
          AND gmt_delete IS NULL
    ) THEN
        RAISE EXCEPTION 'Zhang Wei must be bound to the development admin user';
    END IF;

    RAISE NOTICE
        'Organization demo data ready: source_system_code=%, legal_entities=7, structures=2, units=9, nodes=13, positions=8, employees=5, assignments=8',
        v_source_system;
END
$organization_demo$;

COMMIT;
