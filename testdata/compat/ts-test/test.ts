// Runtime tests for generated .pb.ts files.
// Verifies enum values, name mapping, validation rules, interface type safety,
// and cross-file ES module imports.

import {
  Status,
  StatusName,
  type Address,
  type Person,
} from "../ts/person.entity.pb.ts";

import { type PersonCreate, PersonCreateRules } from "../ts/person.create.pb.ts";
// PersonCreateRules now generated with validate rules from create proto.

import { type PersonUpdateByName, PersonUpdateByNameRules } from "../ts/person.update.pb.ts";

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

import {
  type AllScalars,
  type AllRepeated,
  type TreeNode,
} from "../ts/all_types.entity.pb.ts";

import {
  type AllValidate,
  AllValidateRules,
} from "../ts/all_validate.pb.ts";

import { type AllScalarsCreate } from "../ts/all_types.create.pb.ts";

import { type AllScalarsUpdate, AllRepeatedUpdateRules } from "../ts/all_types.update.pb.ts";

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

// --- Validation rules (person.entity.pb.ts) ---
// Entity proto does not carry validate annotations; rules are in create/update variants.

// --- Validation rules (person.create.pb.ts) ---
// PersonCreateRules inherits constraints from source Person.

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
// status field uses imported Status type from person.entity.pb.ts
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

// --- AllScalars: scalar type mappings ---

const scalars: AllScalars = {
  fSint32: -1,
  fSint64: "-9007199254740993",
  fSfixed32: 0,
  fSfixed64: "0",
  fDouble: 3.14,
  fFixed32: 0,
  fFixed64: "0",
  fUint32: 0,
  fUint64: "0",
  fFloat: 1.5,
  fBytes: "aGVsbG8=",
};
assertEqual(typeof scalars.fSint32, "number", "AllScalars.fSint32 is number");
assertEqual(typeof scalars.fSint64, "string", "AllScalars.fSint64 is string (int64 → string)");
assertEqual(typeof scalars.fDouble, "number", "AllScalars.fDouble is number");
assertEqual(typeof scalars.fBytes, "string", "AllScalars.fBytes is string (bytes → string)");

// --- AllRepeated: repeated field type mappings ---

const repeated: AllRepeated = {
  rSint32: [-1, 0, 1],
  rSfixed32: [0],
  rDouble: [1.1, 2.2],
  rBytes: ["aGVsbG8="],
  rMessage: [{ street: "1 Main", city: "Springfield" }],
  rEnum: [Status.STATUS_ACTIVE, Status.STATUS_INACTIVE],
};
assertEqual(Array.isArray(repeated.rSint32), true, "AllRepeated.rSint32 is array");
assertEqual(repeated.rSint32[0], -1, "AllRepeated.rSint32[0] === -1");
assertEqual(repeated.rMessage[0].city, "Springfield", "AllRepeated.rMessage cross-file Address type works");
assertEqual(repeated.rEnum[0], Status.STATUS_ACTIVE, "AllRepeated.rEnum uses imported Status enum");

// --- AllValidateRules: every constraint type has correct runtime value ---

// uint constraints
assertEqual(AllValidateRules.uGte.type, "integer", "AllValidateRules.uGte.type === integer");
assertEqual(AllValidateRules.uGte.minimum, 1, "AllValidateRules.uGte.minimum === 1");
assertEqual(AllValidateRules.uLte.maximum, 1000, "AllValidateRules.uLte.maximum === 1000");
assert((AllValidateRules.uIn as { enum: readonly number[] }).enum.includes(1), "AllValidateRules.uIn.enum includes 1");
assert((AllValidateRules.uIn as { enum: readonly number[] }).enum.includes(2), "AllValidateRules.uIn.enum includes 2");
assert((AllValidateRules.uIn as { enum: readonly number[] }).enum.includes(3), "AllValidateRules.uIn.enum includes 3");
assert((AllValidateRules.uNotIn as { notIn: readonly number[] }).notIn.includes(0), "AllValidateRules.uNotIn.notIn includes 0");

// float/double constraints
assertEqual((AllValidateRules.fGt as { type: string }).type, "number", "AllValidateRules.fGt.type === number");
assertEqual((AllValidateRules.fGt as { exclusiveMinimum: number }).exclusiveMinimum, 0, "AllValidateRules.fGt.exclusiveMinimum === 0");
assertEqual((AllValidateRules.dLte as { maximum: number }).maximum, 1, "AllValidateRules.dLte.maximum === 1");

// string in/not_in
assertEqual((AllValidateRules.sIn as { type: string }).type, "string", "AllValidateRules.sIn.type === string");
assert((AllValidateRules.sIn as { enum: readonly string[] }).enum.includes("a"), "AllValidateRules.sIn.enum includes a");
assert((AllValidateRules.sIn as { enum: readonly string[] }).enum.includes("b"), "AllValidateRules.sIn.enum includes b");
assert((AllValidateRules.sIn as { enum: readonly string[] }).enum.includes("c"), "AllValidateRules.sIn.enum includes c");
assert((AllValidateRules.sNotIn as { notIn: readonly string[] }).notIn.includes("x"), "AllValidateRules.sNotIn.notIn includes x");
assert((AllValidateRules.sNotIn as { notIn: readonly string[] }).notIn.includes("y"), "AllValidateRules.sNotIn.notIn includes y");

// signed int in
assertEqual((AllValidateRules.iIn as { type: string }).type, "integer", "AllValidateRules.iIn.type === integer");
assert((AllValidateRules.iIn as { enum: readonly number[] }).enum.includes(1), "AllValidateRules.iIn.enum includes 1");
assert((AllValidateRules.iIn as { enum: readonly number[] }).enum.includes(-1), "AllValidateRules.iIn.enum includes -1");

// string uri format
assertEqual((AllValidateRules.sUri as { format: string }).format, "uri", "AllValidateRules.sUri.format === uri");

// optional enum defined_only
assertEqual(AllValidateRules.oStatus.definedOnly, true, "AllValidateRules.oStatus.definedOnly === true");
assert((AllValidateRules.oStatus as { notIn: readonly number[] }).notIn.includes(0), "AllValidateRules.oStatus.notIn includes 0");
assert((AllValidateRules.eStatus as { notIn: readonly number[] }).notIn.includes(0), "AllValidateRules.eStatus.notIn includes 0");
assert((AllValidateRules.eStatus as { notIn: readonly number[] }).notIn.includes(2), "AllValidateRules.eStatus.notIn includes 2");
assert((AllRepeatedUpdateRules.rEnum.items as { notIn: readonly number[] }).notIn.includes(0), "AllRepeatedUpdateRules.rEnum.items.notIn includes 0");

// bytes min/max len
assertEqual((AllValidateRules.bMinmax as { type: string }).type, "string", "AllValidateRules.bMinmax.type === string");
assertEqual((AllValidateRules.bMinmax as { minLength: number }).minLength, 1, "AllValidateRules.bMinmax.minLength === 1");
assertEqual((AllValidateRules.bMinmax as { maxLength: number }).maxLength, 100, "AllValidateRules.bMinmax.maxLength === 100");

// repeated items constraint
assertEqual((AllValidateRules.rItems as { type: string }).type, "array", "AllValidateRules.rItems.type === array");
assertEqual((AllValidateRules.rItems as { minItems: number }).minItems, 1, "AllValidateRules.rItems.minItems === 1");
assertEqual((AllValidateRules.rItems as { maxItems: number }).maxItems, 5, "AllValidateRules.rItems.maxItems === 5");
assertEqual((AllValidateRules.rItems as { items: { minimum: number } }).items.minimum, 0, "AllValidateRules.rItems.items.minimum === 0");
assertEqual((AllValidateRules.rItems as { items: { type: string } }).items.type, "integer", "AllValidateRules.rItems.items.type === integer");

// exclusive bounds for signed int (gt + lt → exclusiveMinimum + exclusiveMaximum)
assertEqual((AllValidateRules.iGtLt as { exclusiveMinimum: number }).exclusiveMinimum, -10, "AllValidateRules.iGtLt.exclusiveMinimum === -10");
assertEqual((AllValidateRules.iGtLt as { exclusiveMaximum: number }).exclusiveMaximum, 10, "AllValidateRules.iGtLt.exclusiveMaximum === 10");

// exclusive bounds for unsigned int (gt + lt → exclusiveMinimum + exclusiveMaximum)
assertEqual((AllValidateRules.uGtLt as { exclusiveMinimum: number }).exclusiveMinimum, 5, "AllValidateRules.uGtLt.exclusiveMinimum === 5");
assertEqual((AllValidateRules.uGtLt as { exclusiveMaximum: number }).exclusiveMaximum, 100, "AllValidateRules.uGtLt.exclusiveMaximum === 100");

// float exclusiveMaximum (lt)
assertEqual((AllValidateRules.fLt as { exclusiveMaximum: number }).exclusiveMaximum, 99.5, "AllValidateRules.fLt.exclusiveMaximum === 99.5");

// double exclusiveMinimum (gt)
assertEqual((AllValidateRules.dGt as { exclusiveMinimum: number }).exclusiveMinimum, -1, "AllValidateRules.dGt.exclusiveMinimum === -1");

// string pattern
assertEqual((AllValidateRules.sPattern as { pattern: string }).pattern, "^[A-Z][a-z]+$", "AllValidateRules.sPattern.pattern === /^[A-Z][a-z]+$/");

// --- AllScalarsCreate: all fields optional (create message) ---

const createAll: AllScalarsCreate = {};
assertEqual(createAll.fSint32, undefined, "AllScalarsCreate all fields optional — fSint32 omitted === undefined");

const createAllFull: AllScalarsCreate = {
  fSint32: -1,
  fSint64: "-1",
  fSfixed32: 0,
  fSfixed64: "0",
  fFixed32: 0,
  fFixed64: "0",
  fUint32: 0,
  fUint64: "0",
  fFloat: 1.0,
  fBytes: "dGVzdA==",
};
assertEqual(createAllFull.fSint32, -1, "AllScalarsCreate.fSint32 assigned correctly");

// --- AllScalarsUpdate: condition field non-optional, rest optional ---

const updateAll: AllScalarsUpdate = { fSint32: 42 };
assertEqual(updateAll.fSint32, 42, "AllScalarsUpdate.fSint32 (condition field) is required and assigned");
assertEqual(updateAll.fSint64, undefined, "AllScalarsUpdate.fSint64 optional — omitted === undefined");

// --- Create/Update validate rule inheritance ---

// PersonCreateRules: nickname is required (in required_fields)
assertEqual(PersonCreateRules.nickname.required, true, "PersonCreateRules.nickname.required === true");
assertEqual(PersonCreateRules.nickname.minLength, 1, "PersonCreateRules.nickname.minLength === 1");
assertEqual(PersonCreateRules.nickname.maxLength, 10, "PersonCreateRules.nickname.maxLength === 10");
// name is optional in create, but inherits constraints from source
assertEqual(PersonCreateRules.name.required, false, "PersonCreateRules.name.required === false");
assertEqual(PersonCreateRules.name.minLength, 1, "PersonCreateRules.name.minLength === 1");
assertEqual(PersonCreateRules.name.maxLength, 100, "PersonCreateRules.name.maxLength === 100");
// age is optional in create
assertEqual(PersonCreateRules.age.required, false, "PersonCreateRules.age.required === false");
assertEqual(PersonCreateRules.age.minimum, 0, "PersonCreateRules.age.minimum === 0");
assertEqual(PersonCreateRules.age.maximum, 150, "PersonCreateRules.age.maximum === 150");
// status is optional in create
assertEqual(PersonCreateRules.status.required, false, "PersonCreateRules.status.required === false");
assertEqual(PersonCreateRules.status.definedOnly, true, "PersonCreateRules.status.definedOnly === true");
// email is optional in create
assertEqual(PersonCreateRules.email.required, false, "PersonCreateRules.email.required === false");
assertEqual(PersonCreateRules.email.format, "email", "PersonCreateRules.email.format === email");
// role is optional in create
assertEqual(PersonCreateRules.role.required, false, "PersonCreateRules.role.required === false");
assert((PersonCreateRules.role as { enum: readonly string[] }).enum.includes("admin"), "PersonCreateRules.role.enum includes admin");
// typeId notIn constraint (inherited from source Person)
assert((PersonCreateRules.typeId as { notIn: readonly number[] }).notIn.includes(0), "PersonCreateRules.typeId.notIn includes 0");
assert((PersonCreateRules.typeId as { notIn: readonly number[] }).notIn.includes(-1), "PersonCreateRules.typeId.notIn includes -1");

// PersonUpdateByNameRules: name is required (condition field)
assertEqual(PersonUpdateByNameRules.name.required, true, "PersonUpdateByNameRules.name.required === true");
assertEqual(PersonUpdateByNameRules.name.minLength, 1, "PersonUpdateByNameRules.name.minLength === 1");
assertEqual(PersonUpdateByNameRules.name.maxLength, 100, "PersonUpdateByNameRules.name.maxLength === 100");
// nickname is optional in update
assertEqual(PersonUpdateByNameRules.nickname.required, false, "PersonUpdateByNameRules.nickname.required === false");
assertEqual(PersonUpdateByNameRules.nickname.minLength, 1, "PersonUpdateByNameRules.nickname.minLength === 1");
assertEqual(PersonUpdateByNameRules.nickname.maxLength, 10, "PersonUpdateByNameRules.nickname.maxLength === 10");

// --- AllValidate: interface type safety ---

const av: AllValidate = {
  uGte: 1,
  uLte: "500",
  uIn: 2,
  uNotIn: 1,
  fGt: 0.1,
  dLte: 0.5,
  sIn: "a",
  sNotIn: "z",
  iIn: 1,
  sUri: "https://example.com",
  bMinmax: "aGVsbG8=",
  rItems: [0, 1, 2],
  iGtLt: 0,
  uGtLt: 50,
  fLt: 50.0,
  dGt: 0.0,
  sPattern: "Hello",
  eStatus: Status.STATUS_ACTIVE,
  rStrIn: ["foo", "bar"],
  rStrNotIn: ["ok"],
  rIntIn: [1, 2],
  rUintNotIn: [1, 2],
};
assertEqual(av.uGte, 1, "AllValidate.uGte assigned correctly");
assertEqual(av.sIn, "a", "AllValidate.sIn assigned correctly");
assertEqual(av.oStatus, undefined, "AllValidate.oStatus optional — omitted === undefined");
assertEqual(av.iGtLt, 0, "AllValidate.iGtLt assigned correctly");
assertEqual(av.sPattern, "Hello", "AllValidate.sPattern assigned correctly");

// --- TreeNode: self-referencing interface type safety ---

const treeLeaf: TreeNode = { value: "leaf", child: { value: "deep", child: { value: "deepest" } } } as TreeNode;
assertEqual(treeLeaf.value, "leaf", "TreeNode.value assigned correctly");
assertEqual(treeLeaf.child.value, "deep", "TreeNode.child.value nested correctly");
assertEqual(treeLeaf.child.child.value, "deepest", "TreeNode.child.child.value deeply nested correctly");

// --- summary ---

assertEqual(passed, 111, "expected exactly 111 assertions before count guard");

console.log(`\n${passed} passed, ${failed} failed`);
if (failed > 0) {
  process.exit(1);
}
