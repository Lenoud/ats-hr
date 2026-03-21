-- migrations/001_init.down.sql
-- Drop triggers
DROP TRIGGER IF EXISTS update_resumes_updated_at ON resumes;
DROP TRIGGER IF EXISTS update_interviews_updated_at ON interviews;
DROP TRIGGER IF EXISTS update_feedbacks_updated_at ON feedbacks;
DROP TRIGGER IF EXISTS update_portfolios_updated_at ON portfolios;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables in reverse order (respecting foreign keys)
DROP TABLE IF EXISTS portfolios;
DROP TABLE IF EXISTS feedbacks;
DROP TABLE IF EXISTS interviews;
DROP TABLE IF EXISTS resumes;