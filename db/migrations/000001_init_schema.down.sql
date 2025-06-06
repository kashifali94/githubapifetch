-- Drop indexes
DROP INDEX IF EXISTS idx_commits_repository_id;
DROP INDEX IF EXISTS idx_commits_date;
DROP INDEX IF EXISTS idx_repositories_name_owner;
 
-- Drop tables
DROP TABLE IF EXISTS commits;
DROP TABLE IF EXISTS repositories; 