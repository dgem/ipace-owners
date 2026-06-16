'use strict';

var crypto = require('crypto');
var database = require('@netlify/database');
var utils = require('./submission-utils');

function rows(result) {
  if (Array.isArray(result)) return result;
  if (result && Array.isArray(result.rows)) return result.rows;
  return [];
}

function jsonParam(value) {
  return JSON.stringify(value == null ? null : value);
}

function getDatabaseConnection() {
  try {
    return database.getDatabase();
  } catch (error) {
    if (error && error.name === 'MissingDatabaseConnectionError') return null;
    if (database.MissingDatabaseConnectionError && error instanceof database.MissingDatabaseConnectionError) return null;
    throw error;
  }
}

function memberId() {
  return 'member_' + crypto.randomUUID();
}

async function saveMemberSnapshot(event, snapshot) {
  var store = utils.getStore(event);
  await store.setJSON('member-snapshots/' + snapshot.identityUserId + '.json', snapshot, {
    metadata: {
      identityUserId: snapshot.identityUserId,
      generatedAt: snapshot.generatedAt,
      type: 'member-snapshot',
    },
  });
}

async function readMemberSnapshot(event, identityUserId) {
  var store = utils.getStore(event);
  return store.get('member-snapshots/' + identityUserId + '.json', { type: 'json' });
}

async function ensureMember(db, data) {
  var id = data.memberId || memberId();
  var identityUserId = data.identityUserId;
  var emailHash = data.emailHash || null;
  var country = data.country || null;
  var displayName = data.displayName || null;
  var relationship = data.relationship || null;

  var result = await db.sql`
    INSERT INTO members (
      id,
      identity_user_id,
      email_hash,
      country,
      display_name,
      relationship_status
    )
    VALUES (
      ${id},
      ${identityUserId},
      ${emailHash},
      ${country},
      ${displayName},
      ${relationship}
    )
    ON CONFLICT (identity_user_id) DO UPDATE SET
      email_hash = COALESCE(EXCLUDED.email_hash, members.email_hash),
      country = COALESCE(EXCLUDED.country, members.country),
      display_name = COALESCE(EXCLUDED.display_name, members.display_name),
      relationship_status = COALESCE(EXCLUDED.relationship_status, members.relationship_status),
      updated_at = NOW()
    RETURNING id
  `;

  return rows(result)[0].id;
}

async function saveJoinRecord(event, record) {
  var db = module.exports.getDatabaseConnection();
  if (!db) return false;

  var member = null;
  var contact = record.contact || {};
  var membership = record.membership || {};
  if (record.identityUserId) {
    member = await ensureMember(db, {
      identityUserId: record.identityUserId,
      emailHash: record.userEmailHash || (contact.email ? utils.emailFingerprint(contact.email) : null),
      country: contact.country,
      displayName: contact.name,
      relationship: membership.relationship,
    });
  }

  await db.sql`
    INSERT INTO join_submissions (
      id,
      member_id,
      identity_user_id,
      email_hash,
      contact_name,
      contact_country,
      relationship_status,
      skills,
      consents,
      review_status,
      verification_level,
      created_at,
      updated_at
    )
    VALUES (
      ${record.id},
      ${member},
      ${record.identityUserId || null},
      ${record.userEmailHash || (contact.email ? utils.emailFingerprint(contact.email) : null)},
      ${contact.name || null},
      ${contact.country || null},
      ${membership.relationship || null},
      ${jsonParam(membership.skills || [])}::jsonb,
      ${jsonParam(record.consents || {})}::jsonb,
      ${(record.review && record.review.status) || 'new'},
      ${(record.review && record.review.verificationLevel) || 'self-reported'},
      ${record.createdAt},
      ${record.updatedAt}
    )
    ON CONFLICT (id) DO NOTHING
  `;

  if (record.identityUserId) {
    await module.exports.regenerateMemberSnapshot(event, {
      identityUserId: record.identityUserId,
      email: contact.email,
    });
  }

  return true;
}

async function saveVehicleRecord(event, record) {
  var db = module.exports.getDatabaseConnection();
  if (!db) return false;

  var vehicle = record.vehicle || {};
  var battery = record.battery || {};
  var member = await ensureMember(db, {
    identityUserId: record.identityUserId,
    emailHash: record.userEmailHash,
  });

  await db.sql`
    INSERT INTO vehicles (
      id,
      member_id,
      identity_user_id,
      vin_hmac,
      vin_last6,
      registration,
      country,
      model_year,
      current_mileage,
      owned_since,
      first_registration_date,
      review_status,
      verification_level,
      created_at,
      updated_at
    )
    VALUES (
      ${record.id},
      ${member},
      ${record.identityUserId},
      ${vehicle.vinHash || null},
      ${vehicle.vinLast6 || null},
      ${vehicle.registration || null},
      ${vehicle.country || null},
      ${vehicle.modelYear || null},
      ${vehicle.mileage},
      ${vehicle.ownedSince || null},
      ${vehicle.firstRegistrationDate || null},
      ${(record.review && record.review.status) || 'new'},
      ${(record.review && record.review.verificationLevel) || 'self-reported'},
      ${record.createdAt},
      ${record.updatedAt}
    )
    ON CONFLICT (id) DO UPDATE SET
      registration = EXCLUDED.registration,
      country = EXCLUDED.country,
      model_year = EXCLUDED.model_year,
      current_mileage = EXCLUDED.current_mileage,
      owned_since = EXCLUDED.owned_since,
      first_registration_date = EXCLUDED.first_registration_date,
      updated_at = EXCLUDED.updated_at
  `;

  if (
    battery.stateOfHealth != null ||
    battery.measuredAt ||
    battery.mileageAtMeasurement != null ||
    battery.source
  ) {
    await db.sql`
      INSERT INTO vehicle_battery_readings (
        id,
        vehicle_id,
        state_of_health,
        measured_at,
        mileage_at_measurement,
        source,
        created_at
      )
      VALUES (
        ${utils.submissionId('battery')},
        ${record.id},
        ${battery.stateOfHealth},
        ${battery.measuredAt || null},
        ${battery.mileageAtMeasurement},
        ${battery.source || null},
        ${record.createdAt}
      )
    `;
  }

  await module.exports.regenerateMemberSnapshot(event, {
    identityUserId: record.identityUserId,
  });

  return true;
}

async function claimJoinSubmissions(db, member, user) {
  var emailHash = user.email ? utils.emailFingerprint(String(user.email).toLowerCase()) : null;
  if (!emailHash) return;

  await db.sql`
    UPDATE join_submissions
    SET
      member_id = ${member.id},
      identity_user_id = ${user.sub},
      updated_at = NOW()
    WHERE email_hash = ${emailHash}
      AND (identity_user_id IS NULL OR identity_user_id = ${user.sub})
  `;
}

async function getOrCreateMemberForUser(db, user) {
  var emailHash = user.email ? utils.emailFingerprint(String(user.email).toLowerCase()) : null;
  var existing = rows(await db.sql`
    SELECT id, identity_user_id
    FROM members
    WHERE identity_user_id = ${user.sub}
    LIMIT 1
  `)[0];

  if (existing) return existing;

  var latestJoin = rows(await db.sql`
    SELECT contact_name, contact_country, relationship_status
    FROM join_submissions
    WHERE email_hash = ${emailHash}
    ORDER BY created_at DESC
    LIMIT 1
  `)[0] || {};

  var id = await ensureMember(db, {
    identityUserId: user.sub,
    emailHash: emailHash,
    country: latestJoin.contact_country,
    displayName: latestJoin.contact_name,
    relationship: latestJoin.relationship_status,
  });

  return { id: id, identity_user_id: user.sub };
}

function mapJoin(row) {
  return {
    id: row.id,
    identityUserId: row.identity_user_id,
    createdAt: row.created_at,
    updatedAt: row.updated_at,
    contact: {
      name: row.contact_name || '',
      country: row.contact_country || '',
    },
    membership: {
      relationship: row.relationship_status || '',
      skills: row.skills || [],
    },
    consents: row.consents || {},
  };
}

function mapVehicle(row) {
  return {
    id: row.id,
    identityUserId: row.identity_user_id,
    createdAt: row.created_at,
    updatedAt: row.updated_at,
    vehicle: {
      vinHash: row.vin_hmac || null,
      vinLast6: row.vin_last6 || '',
      registration: row.registration || '',
      country: row.country || '',
      modelYear: row.model_year || '',
      mileage: row.current_mileage,
      ownedSince: row.owned_since || '',
      firstRegistrationDate: row.first_registration_date || '',
    },
    battery: {
      stateOfHealth: row.state_of_health,
      measuredAt: row.measured_at || '',
      mileageAtMeasurement: row.mileage_at_measurement,
      source: row.source || '',
    },
  };
}

function mapAdminJoin(row) {
  var record = mapJoin(row);
  record.userEmailHash = row.email_hash || null;
  record.review = {
    status: row.review_status || 'new',
    verificationLevel: row.verification_level || 'self-reported',
  };
  return record;
}

function mapAdminVehicle(row) {
  var record = mapVehicle(row);
  record.userEmailHash = row.user_email_hash || null;
  record.review = {
    status: row.review_status || 'new',
    verificationLevel: row.verification_level || 'self-reported',
  };
  return record;
}

async function getAdminData() {
  var db = module.exports.getDatabaseConnection();
  if (!db) return null;

  var joinRecords = rows(await db.sql`
    SELECT
      id,
      identity_user_id,
      email_hash,
      contact_name,
      contact_country,
      relationship_status,
      skills,
      consents,
      review_status,
      verification_level,
      created_at,
      updated_at
    FROM join_submissions
    ORDER BY created_at DESC
  `).map(mapAdminJoin);

  var vehicleRecords = rows(await db.sql`
    SELECT DISTINCT ON (v.id)
      v.id,
      v.identity_user_id,
      m.email_hash AS user_email_hash,
      v.vin_hmac,
      v.vin_last6,
      v.registration,
      v.country,
      v.model_year,
      v.current_mileage,
      v.owned_since,
      v.first_registration_date,
      v.review_status,
      v.verification_level,
      v.created_at,
      v.updated_at,
      b.state_of_health,
      b.measured_at,
      b.mileage_at_measurement,
      b.source
    FROM vehicles v
    LEFT JOIN members m
      ON m.id = v.member_id
    LEFT JOIN vehicle_battery_readings b
      ON b.vehicle_id = v.id
    ORDER BY v.id, b.created_at DESC NULLS LAST
  `).map(mapAdminVehicle);

  return {
    joinRecords: joinRecords,
    vehicleRecords: vehicleRecords,
  };
}

async function regenerateMemberSnapshot(event, user) {
  var db = module.exports.getDatabaseConnection();
  if (!db || !user || !user.identityUserId) return null;

  var member = await getOrCreateMemberForUser(db, {
    sub: user.identityUserId,
    email: user.email,
  });
  await claimJoinSubmissions(db, member, {
    sub: user.identityUserId,
    email: user.email,
  });

  var joinRecords = rows(await db.sql`
    SELECT
      id,
      identity_user_id,
      contact_name,
      contact_country,
      relationship_status,
      skills,
      consents,
      created_at,
      updated_at
    FROM join_submissions
    WHERE member_id = ${member.id}
       OR identity_user_id = ${user.identityUserId}
    ORDER BY created_at DESC
  `).map(mapJoin);

  var vehicleRecords = rows(await db.sql`
    SELECT DISTINCT ON (v.id)
      v.id,
      v.identity_user_id,
      v.vin_hmac,
      v.vin_last6,
      v.registration,
      v.country,
      v.model_year,
      v.current_mileage,
      v.owned_since,
      v.first_registration_date,
      v.created_at,
      v.updated_at,
      b.state_of_health,
      b.measured_at,
      b.mileage_at_measurement,
      b.source
    FROM vehicles v
    LEFT JOIN vehicle_battery_readings b
      ON b.vehicle_id = v.id
    WHERE v.member_id = ${member.id}
       OR v.identity_user_id = ${user.identityUserId}
    ORDER BY v.id, b.created_at DESC NULLS LAST
  `).map(mapVehicle);

  var snapshot = {
    identityUserId: user.identityUserId,
    email: user.email || null,
    joinRecords: joinRecords,
    vehicleRecords: vehicleRecords,
    generatedAt: new Date().toISOString(),
  };

  await db.sql`
    INSERT INTO member_static_snapshots (
      member_id,
      identity_user_id,
      snapshot,
      generated_at
    )
    VALUES (
      ${member.id},
      ${user.identityUserId},
      ${jsonParam(snapshot)}::jsonb,
      NOW()
    )
    ON CONFLICT (member_id) DO UPDATE SET
      identity_user_id = EXCLUDED.identity_user_id,
      snapshot = EXCLUDED.snapshot,
      generated_at = EXCLUDED.generated_at
  `;

  await saveMemberSnapshot(event, snapshot);
  return snapshot;
}

async function getMemberSnapshot(event, user) {
  try {
    var snapshot = await readMemberSnapshot(event, user.sub);
    if (snapshot) return snapshot;
  } catch (_) {
    // Fall through to database regeneration or legacy reads.
  }

  return module.exports.regenerateMemberSnapshot(event, {
    identityUserId: user.sub,
    email: user.email,
  });
}

module.exports = {
  getAdminData: getAdminData,
  getDatabaseConnection: getDatabaseConnection,
  getMemberSnapshot: getMemberSnapshot,
  regenerateMemberSnapshot: regenerateMemberSnapshot,
  saveJoinRecord: saveJoinRecord,
  saveVehicleRecord: saveVehicleRecord,
};
