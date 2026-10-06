-- Development-only seed. Replace the example password before using locally.
INSERT INTO public.admin_user (username, password_hash)
VALUES ('admin', crypt('change-me-locally', gen_salt('bf', 12)))
ON CONFLICT (username) DO NOTHING;
