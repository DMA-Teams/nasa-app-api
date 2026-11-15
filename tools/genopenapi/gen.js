const fs = require("fs");
const path = require("path");

const spec = fs.readFileSync(path.join(__dirname, "..", "..", "api", "openapi.yaml"), "utf8");
if (spec.includes("`")) throw new Error("backtick in openapi.yaml");

process.stdout.write("package httpapi\n\nconst openAPISpec = `" + spec + "`\n");
