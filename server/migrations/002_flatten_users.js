// Run with: mongosh "$MONGODB_URI/siem" migrations/002_flatten_users.js
// User documents created before the fix stored every field except the hash
// under a nested "user" key, so lookups by email never matched. The API also
// applies this automatically at startup. Idempotent.

const database = db.getSiblingDB("siem");

const result = database.users.updateMany(
  { user: { $type: "object" } },
  [{ $replaceWith: { $mergeObjects: ["$$ROOT", "$user"] } }, { $unset: "user" }],
);

print(`Migration 002_flatten_users completed (${result.modifiedCount} document(s) updated)`);
