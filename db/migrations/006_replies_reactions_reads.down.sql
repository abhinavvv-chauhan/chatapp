DROP TABLE IF EXISTS channel_read_states;
DROP TABLE IF EXISTS message_reactions;
ALTER TABLE messages DROP COLUMN IF EXISTS parent_id;
