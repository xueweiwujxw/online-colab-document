CREATE UNIQUE INDEX IF NOT EXISTS users_oidc_subject_unique
    ON users (oidc_subject)
    WHERE oidc_subject IS NOT NULL;
