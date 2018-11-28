CREATE TABLE IF NOT EXISTS release_announcements (
  id BIGSERIAL PRIMARY KEY, 
  published_at TIMESTAMP
);


CREATE TABLE IF NOT EXISTS calls_arranged (
  id BIGSERIAL PRIMARY KEY, 
  arranged_for TIMESTAMP
);
  
CREATE TABLE IF NOT EXISTS monthly_revenue (
  id BIGSERIAL PRIMARY KEY, 
  calculated_at TIMESTAMP,
  projected_monthly_revenue_gbp decimal(12, 2)
);


CREATE TABLE IF NOT EXISTS trials_started (
  id BIGSERIAL PRIMARY KEY, 
  started_at TIMESTAMP
);
