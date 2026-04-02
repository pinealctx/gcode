// Runtime tests for generated .pb.ts files.
// Verifies enum values, name mapping, validation rules, interface type safety,
// and cross-file ES module imports.

import {
  Status,
  StatusName,
  type Address,
  type Person,
  PersonRules,
} from "../ts/person.pb.ts";

import { type PersonCreate } from "../ts/person.create.pb.ts";

import { type PersonUpdateByName } from "../ts/person.update.pb.ts";

import {
  type CreatePersonResponse,
  type GetPersonRequest,
  type GetPersonResponse,
  type UpdatePersonResponse,
  type DeletePersonRequest,
  type DeletePersonResponse,
  GetPersonRequestRules,
  DeletePersonRequestRules,
} from "../ts/person_service.pb.ts";

// --- helpers ---

let passed = 0;
let failed = 0;

function assert(condition: boolean, msg: string): void {
  if (!condition) {
    failed++;
    console.error(`FAIL: ${msg}`);
    process.exitCode = 1;
  } else {
    passed++;
  }
}

function assertEqual<T>(actual: T, expected: T, msg: string): void {
  if (actual !== expected) {
    failed++;
    console.error(`FAIL: ${msg} — expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
    process.exitCode = 1;
  } else {
    passed++;
  }
}

// --- Enum values ---

assertEqual(Status.STATUS_UNSPECIFIED, 0, "Status.STATUS_UNSPECIFIED === 0");
assertEqual(Status.STATUS_ACTIVE, 1, "Status.STATUS_ACTIVE === 1");
assertEqual(Status.STATUS_INACTIVE, 2, "Status.STATUS_INACTIVE === 2");

// --- Name mapping ---

assertEqual(StatusName[Status.STATUS_UNSPECIFIED], "STATUS_UNSPECIFIED", "StatusName[0] === STATUS_UNSPECIFIED");
assertEqual(StatusName[Status.STATUS_ACTIVE], "STATUS_ACTIVE", "StatusName[1] === STATUS_ACTIVE");
assertEqual(StatusName[Status.STATUS_INACTIVE], "STATUS_INACTIVE", "StatusName[2] === STATUS_INACTIVE");

// --- Validation rules (person.pb.ts) ---

assertEqual(PersonRules.name.minLength, 1, "PersonRules.name.minLength === 1");
assertEqual(PersonRules.name.maxLength, 100, "PersonRules.name.maxLength === 100");
assertEqual(PersonRules.age.minimum, 0, "PersonRules.age.minimum === 0");
assertEqual(PersonRules.age.maximum, 150, "PersonRules.age.maximum === 150");
assertEqual(PersonRules.status.definedOnly, true, "PersonRules.status.definedOnly === true");
assertEqual(PersonRules.address.required, true, "PersonRules.address.required === true");
assertEqual(PersonRules.scores.minItems, 1, "PersonRules.scores.minItems === 1");
assertEqual(PersonRules.scores.maxItems, 100, "PersonRules.scores.maxItems === 100");
assertEqual(PersonRules.tags.minItems, 1, "PersonRules.tags.minItems === 1");
assertEqual(PersonRules.email.format, "email", "PersonRules.email.format === email");

// enum constraint
assert((PersonRules.role as { enum: readonly string[] }).enum.includes("admin"), "PersonRules.role.enum includes admin");
assert((PersonRules.role as { enum: readonly string[] }).enum.includes("user"), "PersonRules.role.enum includes user");
assert((PersonRules.role as { enum: readonly string[] }).enum.includes("guest"), "PersonRules.role.enum includes guest");

// notIn constraint
assert((PersonRules.typeId as { notIn: readonly number[] }).notIn.includes(0), "PersonRules.typeId.notIn includes 0");
assert((PersonRules.typeId as { notIn: readonly number[] }).notIn.includes(-1), "PersonRules.typeId.notIn includes -1");

// --- Validation rules (person_service.pb.ts) ---

assertEqual(GetPersonRequestRules.id.minLength, 1, "GetPersonRequestRules.id.minLength === 1");
assertEqual(GetPersonRequestRules.id.maxLength, 64, "GetPersonRequestRules.id.maxLength === 64");
assertEqual(DeletePersonRequestRules.id.minLength, 1, "DeletePersonRequestRules.id.minLength === 1");
assertEqual(DeletePersonRequestRules.id.maxLength, 64, "DeletePersonRequestRules.id.maxLength === 64");

// --- Interface type safety (compile-time + runtime) ---

const address: Address = { street: "123 Main St", city: "Springfield" };
assertEqual(address.street, "123 Main St", "Address.street assigned correctly");
assertEqual(address.city, "Springfield", "Address.city assigned correctly");

const person: Person = {
  name: "Alice",
  age: 30,
  active: true,
  status: Status.STATUS_ACTIVE,
  address,
  scores: [95, 87],
  tags: ["dev"],
  rating: 4.5,
  createdAt: "2024-01-01",
  avatar: "aW1hZ2U=",
  email: "alice@example.com",
  role: "admin",
  typeId: 1,
};
assertEqual(person.name, "Alice", "Person.name assigned correctly");
assertEqual(person.status, Status.STATUS_ACTIVE, "Person.status is Status enum value");

// --- Scalar type mappings (proto → TS) ---

assertEqual(typeof person.createdAt, "string", "int64 field renders as string");
assertEqual(typeof person.avatar, "string", "bytes field renders as string");
assertEqual(typeof person.active, "boolean", "bool field renders as boolean");
assertEqual(typeof person.rating, "number", "float field renders as number");

// optional fields can be omitted
const personPartial: Person = {
  name: "Bob",
  age: 25,
  active: false,
  status: Status.STATUS_INACTIVE,
  address: { street: "456 Oak Ave", city: "Portland" },
  scores: [],
  tags: [],
  rating: 3.0,
  createdAt: "2024-06-15",
  avatar: "Ymlu",
  email: "bob@example.com",
  role: "user",
  typeId: 2,
};
assertEqual(personPartial.nickname, undefined, "Person.nickname omitted === undefined");

// --- Cross-file imports ---

const create: PersonCreate = {
  nickname: "Al",
};
// status field uses imported Status type from person.pb.ts
const createWithStatus: PersonCreate = {
  nickname: "Bob",
  status: Status.STATUS_ACTIVE,
};
assertEqual(createWithStatus.status, Status.STATUS_ACTIVE, "PersonCreate.status cross-file import works");

const update: PersonUpdateByName = {
  name: "Alice",
  status: Status.STATUS_INACTIVE,
};
assertEqual(update.name, "Alice", "PersonUpdateByName.name assigned correctly");
assertEqual(update.status, Status.STATUS_INACTIVE, "PersonUpdateByName.status cross-file import works");

const getReq: GetPersonRequest = { id: "abc123" };
assertEqual(getReq.id, "abc123", "GetPersonRequest.id assigned correctly");

const createResp: CreatePersonResponse = { id: "new-id" };
assertEqual(createResp.id, "new-id", "CreatePersonResponse.id assigned correctly");

const getResp: GetPersonResponse = { name: "Alice", age: 30 };
assertEqual(getResp.name, "Alice", "GetPersonResponse.name assigned correctly");

const updateResp: UpdatePersonResponse = { ok: true };
assertEqual(updateResp.ok, true, "UpdatePersonResponse.ok assigned correctly");

const delReq: DeletePersonRequest = { id: "xyz789" };
assertEqual(delReq.id, "xyz789", "DeletePersonRequest.id assigned correctly");

const delResp: DeletePersonResponse = { ok: false };
assertEqual(delResp.ok, false, "DeletePersonResponse.ok assigned correctly");

// --- summary ---

assertEqual(passed, 43, "expected exactly 43 assertions before count guard");

console.log(`\n${passed} passed, ${failed} failed`);
if (failed > 0) {
  process.exit(1);
}
