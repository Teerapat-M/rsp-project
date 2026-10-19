ALTER TABLE rooms DROP CONSTRAINT IF EXISTS rooms_passcode_digits_chk;
ALTER TABLE rooms ADD CONSTRAINT rooms_passcode_check CHECK (passcode <> '');

DROP TABLE IF EXISTS password_reset_tokens;
DROP TABLE IF EXISTS refresh_tokens;

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_guest_chk;
ALTER TABLE users ADD CONSTRAINT users_guest_email_chk CHECK (
    (is_guest AND email IS NULL) OR (NOT is_guest AND email IS NOT NULL)
);

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_password_pair_chk;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_username_format_chk;
DROP INDEX IF EXISTS users_username_lower_idx;
ALTER TABLE users DROP COLUMN IF EXISTS password_hash;
ALTER TABLE users DROP COLUMN IF EXISTS username;
