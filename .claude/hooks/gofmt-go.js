const { execFileSync } = require('node:child_process');

function readStdin() {
  return new Promise((resolve) => {
    let data = '';
    process.stdin.on('data', (chunk) => { data += chunk; });
    process.stdin.on('end', () => resolve(data));
  });
}

function emit(message) {
  process.stdout.write(JSON.stringify({
    hookSpecificOutput: { hookEventName: 'PostToolUse', additionalContext: message },
  }));
}

(async () => {
  let filePath;
  try {
    const payload = JSON.parse(await readStdin());
    filePath = payload?.tool_input?.file_path;
  } catch {
    process.exit(0);
  }

  if (typeof filePath !== 'string' || !filePath.endsWith('.go')) {
    process.exit(0);
  }

  try {
    execFileSync('gofmt', ['-w', filePath], { stdio: ['ignore', 'ignore', 'pipe'] });
  } catch (error) {
    if (error.code === 'ENOENT') {
      emit(`gofmt no está instalado: ${filePath} quedó sin formatear. Instalar Go (ver .claude/CLAUDE.md, sección Herramientas).`);
    } else {
      const detail = error.stderr ? error.stderr.toString().trim() : error.message;
      emit(`gofmt falló en ${filePath}: ${detail}`);
    }
  }
  process.exit(0);
})();
