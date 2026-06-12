ALTER TABLE contacts DROP CONSTRAINT contacts_pkey;
ALTER TABLE contacts DROP COLUMN user_id;
ALTER TABLE contacts ADD PRIMARY KEY (name);
