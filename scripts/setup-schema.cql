-- Create keyspace
CREATE KEYSPACE IF NOT EXISTS reproduction 
WITH replication = {
  'class': 'SimpleStrategy',
  'replication_factor': 1
};

USE reproduction;

-- Create table with composite primary key
CREATE TABLE IF NOT EXISTS platform_users (
  org_id TEXT,
  user_id TEXT,
  user_data TEXT,
  version INT,
  created_at TIMESTAMP,
  PRIMARY KEY (org_id, user_id)
);

-- Insert test data
INSERT INTO platform_users (org_id, user_id, user_data, version, created_at)
VALUES ('org-1', 'user-1', '{"name":"John"}', 1, toTimestamp(now()));

INSERT INTO platform_users (org_id, user_id, user_data, version, created_at)
VALUES ('org-1', 'user-2', '{"name":"Jane"}', 1, toTimestamp(now()));

INSERT INTO platform_users (org_id, user_id, user_data, version, created_at)
VALUES ('org-1', 'user-3', '{"name":"Bob"}', 1, toTimestamp(now()));

-- This should work in cqlsh
SELECT COUNT(*) FROM platform_users WHERE org_id = 'org-1';
-- Expected: 3

DELETE FROM platform_users WHERE org_id = 'org-1';

SELECT COUNT(*) FROM platform_users WHERE org_id = 'org-1';
-- Expected: 0