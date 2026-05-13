import assert from "node:assert/strict";
import { mkdir, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";
import test from "node:test";

import { generateTypesFromSwaggerFile } from "./generate-swagger-types.mjs";

test("generates deterministic TypeScript contracts from committed Swagger schema", async () => {
  const repoRoot = path.resolve(import.meta.dirname, "../../..");
  const outputDir = await mkdir(path.join(tmpdir(), `tachigo-shared-types-${Date.now()}`), {
    recursive: true,
  });
  const outputFile = path.join(outputDir, "index.ts");

  try {
    await generateTypesFromSwaggerFile({
      inputFile: path.join(repoRoot, "services/api/docs/swagger.json"),
      outputFile,
    });

    const generated = await readFile(outputFile, "utf8");

    assert.match(generated, /export interface HandlersAuthResponse \{/);
    assert.match(generated, /tokens\?: HandlersBrowserTokenPair;/);
    assert.match(generated, /export interface ServicesLoginInput \{/);
    assert.match(generated, /email\?: string;/);
    assert.match(generated, /export interface ApiOperations \{/);
    assert.match(generated, /"POST \/auth\/login": \{/);
    assert.match(generated, /requestBody: ServicesLoginInput;/);
    assert.match(generated, /response: HandlersResponse & \{\s+data\?: HandlersAuthResponse;/);
  } finally {
    await rm(outputDir, { recursive: true, force: true });
  }
});
