ALTER TABLE proposals DROP COLUMN client_phone, DROP COLUMN client_name;
DROP INDEX leads_phone_idx;
ALTER TABLE leads DROP COLUMN merged_into;
