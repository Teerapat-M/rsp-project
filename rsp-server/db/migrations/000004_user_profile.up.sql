-- Avatar/color on users so JWT sessions can restore the guest picker
-- without keeping that state only in the hub.

ALTER TABLE users
    ADD COLUMN avatar TEXT NOT NULL DEFAULT 'rock',
    ADD COLUMN color TEXT NOT NULL DEFAULT '#9DB89A';
