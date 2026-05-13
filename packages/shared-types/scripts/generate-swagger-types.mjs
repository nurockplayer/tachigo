import { mkdir, readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import process from "node:process";

const HTTP_METHODS = ["get", "post", "put", "patch", "delete", "options", "head"];
const GENERATED_HEADER = `/* eslint-disable */\n// Generated from services/api/docs/swagger.json. Do not edit by hand.\n\n`;

export async function generateTypesFromSwaggerFile({ inputFile, outputFile }) {
  const swagger = JSON.parse(await readFile(inputFile, "utf8"));
  const source = generateTypes(swagger);

  await mkdir(path.dirname(outputFile), { recursive: true });
  await writeFile(outputFile, source);
}

export async function checkTypesFromSwaggerFile({ inputFile, outputFile }) {
  const swagger = JSON.parse(await readFile(inputFile, "utf8"));
  const expected = generateTypes(swagger);
  const current = await readFile(outputFile, "utf8").catch(() => "");

  if (current !== expected) {
    throw new Error(`${outputFile} is out of date. Run pnpm api:types:generate.`);
  }
}

export function generateTypes(swagger) {
  const definitions = swagger.definitions ?? {};
  const lines = [GENERATED_HEADER.trimEnd(), ""];

  for (const name of Object.keys(definitions).sort()) {
    lines.push(renderDefinition(name, definitions[name]), "");
  }

  lines.push(renderOperations(swagger.paths ?? {}));
  return `${lines.join("\n")}\n`;
}

function renderDefinition(name, schema) {
  const typeName = refName(name);

  if (schema.enum) {
    return `export type ${typeName} = ${renderEnum(schema.enum)};`;
  }

  if (schema.type === "object" || schema.properties || schema.allOf) {
    const objectType = schemaToType(schema, 0);
    if (objectType.startsWith("{\n")) {
      return `export interface ${typeName} ${objectType}`;
    }
    return `export type ${typeName} = ${objectType};`;
  }

  return `export type ${typeName} = ${schemaToType(schema, 0)};`;
}

function renderOperations(paths) {
  const lines = ["export interface ApiOperations {"];

  for (const route of Object.keys(paths).sort()) {
    const pathItem = paths[route] ?? {};
    for (const method of HTTP_METHODS) {
      const operation = pathItem[method];
      if (!operation) continue;
      lines.push(`  "${method.toUpperCase()} ${route}": ${renderOperation(operation, 2)};`);
    }
  }

  lines.push("}");
  return lines.join("\n");
}

function renderOperation(operation, indent) {
  const bodyParameter = (operation.parameters ?? []).find((parameter) => parameter.in === "body");
  const pathParameters = (operation.parameters ?? []).filter((parameter) => parameter.in === "path");
  const queryParameters = (operation.parameters ?? []).filter((parameter) => parameter.in === "query");
  const responseType = schemaToType(selectSuccessResponseSchema(operation.responses ?? {}), indent + 2);
  const lines = ["{"];

  if (bodyParameter?.schema) {
    const optional = bodyParameter.required ? "" : "?";
    lines.push(`${spaces(indent + 2)}requestBody${optional}: ${schemaToType(bodyParameter.schema, indent + 2)};`);
  }

  if (pathParameters.length > 0) {
    lines.push(`${spaces(indent + 2)}pathParams: ${renderParameterObject(pathParameters, indent + 2)};`);
  }

  if (queryParameters.length > 0) {
    lines.push(`${spaces(indent + 2)}queryParams: ${renderParameterObject(queryParameters, indent + 2)};`);
  }

  lines.push(`${spaces(indent + 2)}response: ${responseType};`);
  lines.push(`${spaces(indent)}}`);

  return lines.join("\n");
}

function renderParameterObject(parameters, indent) {
  const lines = ["{"];
  for (const parameter of parameters.sort((a, b) => a.name.localeCompare(b.name))) {
    const optional = parameter.required ? "" : "?";
    lines.push(`${spaces(indent + 2)}${propertyKey(parameter.name)}${optional}: ${schemaToType(parameter, indent + 2)};`);
  }
  lines.push(`${spaces(indent)}}`);
  return lines.join("\n");
}

function selectSuccessResponseSchema(responses) {
  const status = Object.keys(responses)
    .filter((key) => key.startsWith("2"))
    .sort()[0];

  return responses[status]?.schema ?? {};
}

function schemaToType(schema = {}, indent) {
  if (schema.$ref) return refName(schema.$ref.split("/").at(-1));
  if (schema.allOf) return schema.allOf.map((item) => schemaToType(item, indent)).join(" & ");
  if (schema.enum) return renderEnum(schema.enum);

  if (schema.type === "array") {
    return `Array<${schemaToType(schema.items ?? {}, indent)}>`;
  }

  if (schema.type === "object" || schema.properties) {
    return renderObjectType(schema, indent);
  }

  if (schema.additionalProperties) {
    return `Record<string, ${schemaToType(schema.additionalProperties, indent)}>`;
  }

  switch (schema.type) {
    case "integer":
    case "number":
      return "number";
    case "boolean":
      return "boolean";
    case "string":
      return "string";
    default:
      return "unknown";
  }
}

function renderObjectType(schema, indent) {
  const properties = schema.properties ?? {};
  const required = new Set(schema.required ?? []);
  const names = Object.keys(properties).sort();

  if (names.length === 0) {
    return "Record<string, unknown>";
  }

  const lines = ["{"];
  for (const name of names) {
    const optional = required.has(name) ? "" : "?";
    lines.push(`${spaces(indent + 2)}${propertyKey(name)}${optional}: ${schemaToType(properties[name], indent + 2)};`);
  }
  lines.push(`${spaces(indent)}}`);
  return lines.join("\n");
}

function renderEnum(values) {
  return values.map((value) => JSON.stringify(value)).join(" | ");
}

function refName(rawName) {
  return rawName
    .split(/[^a-zA-Z0-9]+/)
    .filter(Boolean)
    .map((part) => `${part.charAt(0).toUpperCase()}${part.slice(1)}`)
    .join("");
}

function propertyKey(name) {
  return /^[a-zA-Z_$][a-zA-Z0-9_$]*$/.test(name) ? name : JSON.stringify(name);
}

function spaces(count) {
  return " ".repeat(Math.max(0, count));
}

function parseArgs(argv) {
  const [command, ...rest] = argv;
  const options = { command };

  for (let index = 0; index < rest.length; index += 2) {
    options[rest[index].replace(/^--/, "")] = rest[index + 1];
  }

  if (!["generate", "check"].includes(options.command) || !options.input || !options.output) {
    throw new Error("Usage: generate-swagger-types.mjs <generate|check> --input <swagger.json> --output <index.ts>");
  }

  return options;
}

if (import.meta.url === `file://${process.argv[1]}`) {
  const options = parseArgs(process.argv.slice(2));
  const task =
    options.command === "generate"
      ? generateTypesFromSwaggerFile({ inputFile: options.input, outputFile: options.output })
      : checkTypesFromSwaggerFile({ inputFile: options.input, outputFile: options.output });

  task.catch((error) => {
    console.error(error.message);
    process.exitCode = 1;
  });
}
