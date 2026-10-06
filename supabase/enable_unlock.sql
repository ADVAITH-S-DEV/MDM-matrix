-- SQL Editor deployment patch. Run before deploying Unlock.
-- Preserves command records and existing RLS/grants.
BEGIN;
ALTER TABLE public.commands DROP CONSTRAINT IF EXISTS commands_type_check;
ALTER TABLE public.commands ADD CONSTRAINT commands_type_check
  CHECK (type IN ('lock', 'unlock', 'wipe', 'update_policy'));
COMMIT;
