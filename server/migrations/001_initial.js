// Run with: mongosh "$MONGODB_URI/siem" migrations/001_initial.js
// This migration is intentionally idempotent and safe to run again.

const database = db.getSiblingDB("siem");

[
  "companies",
  "users",
  "addresses",
  "events",
  "alerts",
  "invitations",
  "refresh_tokens",
  "counters",
  "password_resets",
].forEach((collection) => {
  if (!database.getCollectionNames().includes(collection)) {
    database.createCollection(collection);
  }
});

database.companies.createIndex({ name: 1 }, { unique: true, name: "companies_name_unique" });
database.users.createIndex({ email: 1 }, { unique: true, name: "users_email_unique" });
database.users.createIndex({ companyId: 1, role: 1 }, { name: "users_company_role" });
database.users.createIndex({ companyId: 1, email: 1 }, { unique: true, name: "users_company_email_unique" });
database.addresses.createIndex({ companyId: 1, value: 1 }, { unique: true, name: "addresses_company_value_unique" });
database.events.createIndex({ companyId: 1, timestamp: -1 }, { name: "events_company_timestamp" });
database.events.createIndex({ companyId: 1, severity: 1, timestamp: -1 }, { name: "events_company_severity_timestamp" });
database.events.createIndex({ companyId: 1, ip: 1, timestamp: -1 }, { name: "events_company_ip_timestamp" });
database.alerts.createIndex({ companyId: 1, status: 1, createdAt: -1 }, { name: "alerts_company_status_created" });
database.alerts.createIndex({ companyId: 1, severity: 1, createdAt: -1 }, { name: "alerts_company_severity_created" });
// Only the SHA-256 of invitation and refresh tokens is stored.
database.invitations.createIndex({ tokenHash: 1 }, { unique: true, name: "invitations_token_hash_unique" });
database.invitations.createIndex({ companyId: 1, id: 1 }, { name: "invitations_company_id" });
database.invitations.createIndex({ companyId: 1, email: 1 }, { name: "invitations_company_email" });
database.invitations.createIndex({ expiresAt: 1 }, { expireAfterSeconds: 0, name: "invitations_expiry" });
database.refresh_tokens.createIndex({ tokenHash: 1 }, { unique: true, name: "refresh_tokens_hash_unique" });
database.refresh_tokens.createIndex({ userId: 1 }, { name: "refresh_tokens_user" });
database.refresh_tokens.createIndex({ expiresAt: 1 }, { expireAfterSeconds: 0, name: "refresh_tokens_expiry" });
database.password_resets.createIndex({ tokenHash: 1 }, { unique: true, name: "password_resets_hash_unique" });
database.password_resets.createIndex({ userId: 1, used: 1 }, { name: "password_resets_user" });
database.password_resets.createIndex({ expiresAt: 1 }, { expireAfterSeconds: 0, name: "password_resets_expiry" });

print("Migration 001_initial completed");
