-- Drop triggers
DROP TRIGGER IF EXISTS trigger_webhook_queue_updated_at ON webhook_queue;
DROP TRIGGER IF EXISTS trigger_webhooks_updated_at ON webhooks;

-- Drop function
DROP FUNCTION IF EXISTS update_webhook_updated_at();

-- Drop tables (cascade will handle foreign keys)
DROP TABLE IF EXISTS webhook_queue;
DROP TABLE IF EXISTS webhooks;
