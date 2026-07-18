import { readFile } from "node:fs/promises";
import { createRequire } from "node:module";
import { hash } from "./index.js";

const fixture=JSON.parse(await readFile("../../conformance/fixtures/canonical.json","utf8")) as {value:unknown;sha256:string};
const digest=hash(fixture.value);
if(digest!==fixture.sha256)throw new Error(`canonical digest ${digest} != ${fixture.sha256}`);
const schema=JSON.parse(await readFile("../../specification/schemas/v0.1/event.schema.json","utf8"));
const validEvent=JSON.parse(await readFile("../../conformance/fixtures/valid-event.json","utf8"));
const invalidEvent=JSON.parse(await readFile("../../conformance/fixtures/invalid-event.json","utf8"));
const require=createRequire(import.meta.url);
const Ajv2020=require("ajv/dist/2020").default;
const addFormats=require("ajv-formats").default;
const ajv=new Ajv2020({strict:true});addFormats(ajv);const validate=ajv.compile(schema);
if(!validate(validEvent))throw new Error(`valid schema fixture rejected: ${JSON.stringify(validate.errors)}`);
if(validate(invalidEvent))throw new Error("invalid schema fixture accepted");
process.stdout.write("typescript-conformance-ok\n");
