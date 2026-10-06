CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS public.admin_user (
    username text PRIMARY KEY,
    password_hash text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.devices (
    id text PRIMARY KEY,
    name text NOT NULL,
    token text NOT NULL UNIQUE,
    status text NOT NULL DEFAULT 'offline' CHECK (status IN ('online', 'offline')),
    battery integer NOT NULL DEFAULT 0 CHECK (battery BETWEEN 0 AND 100),
    last_seen timestamptz NOT NULL DEFAULT now(),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.commands (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id text NOT NULL REFERENCES public.devices(id) ON DELETE CASCADE,
    type text NOT NULL CHECK (type IN ('lock', 'wipe', 'update_policy')),
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'delivered', 'completed', 'failed')),
    created_at timestamptz NOT NULL DEFAULT now(),
    delivered_at timestamptz,
    completed_at timestamptz,
    failure_reason text
);

ALTER TABLE public.commands ADD COLUMN IF NOT EXISTS delivered_at timestamptz;
ALTER TABLE public.commands ADD COLUMN IF NOT EXISTS failure_reason text;
ALTER TABLE public.commands DROP CONSTRAINT IF EXISTS commands_status_check;
ALTER TABLE public.commands ADD CONSTRAINT commands_status_check
    CHECK (status IN ('pending', 'delivered', 'completed', 'failed'));

CREATE TABLE IF NOT EXISTS public.command_events (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    command_id uuid NOT NULL REFERENCES public.commands(id) ON DELETE CASCADE,
    device_id text NOT NULL REFERENCES public.devices(id) ON DELETE CASCADE,
    activity text NOT NULL CHECK (activity IN ('created', 'queued', 'delivered', 'acknowledged', 'completed', 'failed')),
    occurred_at timestamptz NOT NULL DEFAULT now(),
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_commands_device_status_created
    ON public.commands (device_id, status, created_at);
CREATE INDEX IF NOT EXISTS idx_commands_created_at
    ON public.commands (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_command_events_command_time
    ON public.command_events (command_id, occurred_at);

ALTER TABLE public.admin_user ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.devices ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.commands ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.command_events ENABLE ROW LEVEL SECURITY;

REVOKE ALL ON public.admin_user, public.devices, public.commands, public.command_events FROM anon, authenticated;
REVOKE ALL ON SEQUENCE public.command_events_id_seq FROM anon, authenticated;

COMMENT ON TABLE public.command_events IS 'Append-only command lifecycle event log suitable for process-mining analysis.';
