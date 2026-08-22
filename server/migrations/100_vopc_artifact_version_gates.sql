-- Structured, server-verifiable metadata gates. This does not claim to verify
-- an external repository or object store. Existing rows remain non-usable.
ALTER TABLE vopc_artifact_versions ADD COLUMN status TEXT NOT NULL DEFAULT 'invalid';
ALTER TABLE vopc_artifact_versions ADD COLUMN intended_stage TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_vopc_artifact_version_gate ON vopc_artifact_versions(status,intended_stage,artifact_id);
