-- migrations/001_init.up.sql
-- 简历表
CREATE TABLE IF NOT EXISTS resumes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(100) NOT NULL,
    email           VARCHAR(100),
    phone           VARCHAR(20),
    source          VARCHAR(50),
    file_url        TEXT,
    parsed_data     JSONB,
    status          VARCHAR(20) DEFAULT 'pending',
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 面试记录表
CREATE TABLE IF NOT EXISTS interviews (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resume_id       UUID NOT NULL REFERENCES resumes(id) ON DELETE CASCADE,
    round           INT DEFAULT 1,
    interviewer     VARCHAR(100),
    scheduled_at    TIMESTAMP WITH TIME ZONE,
    status          VARCHAR(20) DEFAULT 'scheduled',
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 面评表
CREATE TABLE IF NOT EXISTS feedbacks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    interview_id    UUID NOT NULL REFERENCES interviews(id) ON DELETE CASCADE,
    rating          INT CHECK (rating >= 1 AND rating <= 5),
    content         TEXT,
    recommendation  VARCHAR(20),
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 作品集表
CREATE TABLE IF NOT EXISTS portfolios (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resume_id       UUID NOT NULL REFERENCES resumes(id) ON DELETE CASCADE,
    title           VARCHAR(200),
    description     TEXT,
    file_url        TEXT,
    file_type       VARCHAR(50),
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 索引
CREATE INDEX idx_resumes_status ON resumes(status);
CREATE INDEX idx_resumes_source ON resumes(source);
CREATE INDEX idx_resumes_created_at ON resumes(created_at DESC);
CREATE INDEX idx_resumes_email ON resumes(email);

CREATE INDEX idx_interviews_resume_id ON interviews(resume_id);
CREATE INDEX idx_interviews_status ON interviews(status);
CREATE INDEX idx_interviews_scheduled_at ON interviews(scheduled_at);

CREATE INDEX idx_feedbacks_interview_id ON feedbacks(interview_id);

CREATE INDEX idx_portfolios_resume_id ON portfolios(resume_id);

-- 更新时间触发器函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 为每个表添加更新时间触发器
CREATE TRIGGER update_resumes_updated_at BEFORE UPDATE ON resumes
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_interviews_updated_at BEFORE UPDATE ON interviews
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_feedbacks_updated_at BEFORE UPDATE ON feedbacks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_portfolios_updated_at BEFORE UPDATE ON portfolios
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();