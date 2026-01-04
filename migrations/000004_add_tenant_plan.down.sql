DROP INDEX IF EXISTS idx_tenants_plan_id;
ALTER TABLE tenants DROP COLUMN IF EXISTS plan_id;
