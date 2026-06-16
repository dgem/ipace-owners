CREATE TABLE members (
  id TEXT PRIMARY KEY,
  identity_user_id TEXT NOT NULL UNIQUE,
  email_hash TEXT,
  country TEXT,
  display_name TEXT,
  relationship_status TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE join_submissions (
  id TEXT PRIMARY KEY,
  member_id TEXT REFERENCES members(id) ON DELETE SET NULL,
  identity_user_id TEXT,
  email_hash TEXT,
  contact_name TEXT,
  contact_country TEXT,
  relationship_status TEXT,
  skills JSONB NOT NULL DEFAULT '[]'::jsonb,
  consents JSONB NOT NULL DEFAULT '{}'::jsonb,
  review_status TEXT NOT NULL DEFAULT 'new',
  verification_level TEXT NOT NULL DEFAULT 'self-reported',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE vehicles (
  id TEXT PRIMARY KEY,
  member_id TEXT NOT NULL REFERENCES members(id) ON DELETE CASCADE,
  identity_user_id TEXT NOT NULL,
  vin_hmac TEXT,
  vin_last6 TEXT,
  registration TEXT,
  country TEXT,
  model_year TEXT,
  current_mileage INTEGER,
  owned_since DATE,
  first_registration_date DATE,
  review_status TEXT NOT NULL DEFAULT 'new',
  verification_level TEXT NOT NULL DEFAULT 'self-reported',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT vehicles_has_identifier CHECK (vin_hmac IS NOT NULL OR registration IS NOT NULL)
);

CREATE TABLE vehicle_battery_readings (
  id TEXT PRIMARY KEY,
  vehicle_id TEXT NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
  state_of_health NUMERIC(5, 2),
  measured_at DATE,
  mileage_at_measurement INTEGER,
  source TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE evidence_files (
  id TEXT PRIMARY KEY,
  member_id TEXT NOT NULL REFERENCES members(id) ON DELETE CASCADE,
  vehicle_id TEXT REFERENCES vehicles(id) ON DELETE SET NULL,
  blob_key TEXT NOT NULL,
  original_filename TEXT,
  content_type TEXT,
  byte_size BIGINT,
  evidence_type TEXT,
  review_status TEXT NOT NULL DEFAULT 'new',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE member_static_snapshots (
  member_id TEXT PRIMARY KEY REFERENCES members(id) ON DELETE CASCADE,
  identity_user_id TEXT NOT NULL UNIQUE,
  snapshot JSONB NOT NULL,
  generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE public_stats_snapshots (
  id TEXT PRIMARY KEY,
  snapshot JSONB NOT NULL,
  generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  published_by_identity_user_id TEXT
);

CREATE TABLE audit_events (
  id TEXT PRIMARY KEY,
  actor_identity_user_id TEXT,
  action TEXT NOT NULL,
  subject_type TEXT NOT NULL,
  subject_id TEXT,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_join_submissions_identity_user_id ON join_submissions(identity_user_id);
CREATE INDEX idx_join_submissions_review_status ON join_submissions(review_status);
CREATE INDEX idx_members_identity_user_id ON members(identity_user_id);
CREATE INDEX idx_vehicles_member_id ON vehicles(member_id);
CREATE INDEX idx_vehicles_identity_user_id ON vehicles(identity_user_id);
CREATE INDEX idx_vehicles_vin_hmac ON vehicles(vin_hmac);
CREATE INDEX idx_vehicles_review_status ON vehicles(review_status);
CREATE INDEX idx_vehicle_battery_readings_vehicle_id ON vehicle_battery_readings(vehicle_id);
CREATE INDEX idx_evidence_files_vehicle_id ON evidence_files(vehicle_id);
CREATE INDEX idx_public_stats_snapshots_generated_at ON public_stats_snapshots(generated_at DESC);
