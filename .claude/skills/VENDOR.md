# Skills de terceros incluidas en este proyecto

## samber/cc-skills-golang

- Repositorio: https://github.com/samber/cc-skills-golang
- Commit: `d68d80aec298e1b74b37c2a472c002b5217248b3` (clonado el 2026-11-14)
- Licencia: MIT, copia en `LICENSE-samber-cc-skills-golang`
- Autor: Samuel Berthe. El README declara que las skills fueron editadas, probadas y revisadas por
  una persona, y publica evaluaciones contra una línea base.

### Skills incluidas (20)

`golang-how-to` (enrutador), `golang-code-style`, `golang-naming`, `golang-structs-interfaces`,
`golang-error-handling`, `golang-concurrency`, `golang-context`, `golang-project-layout`,
`golang-design-patterns`, `golang-dependency-injection`, `golang-safety`, `golang-security`,
`golang-observability`, `golang-performance`, `golang-benchmark`, `golang-testing`, `golang-lint`,
`golang-modernize`, `golang-gopls`, `golang-troubleshooting`.

### Excluidas, y por qué

- Librerías que el proyecto no usa: `golang-samber-*`, `golang-spf13-cobra`, `golang-spf13-viper`,
  `golang-google-wire`, `golang-uber-dig`, `golang-uber-fx`, `golang-stretchr-testify`.
- Tecnologías fuera de alcance: `golang-grpc`, `golang-graphql`, `golang-database` (SQL),
  `golang-cli`, `golang-swagger`.
- Temas que no aportan al MVP: `golang-documentation`, `golang-refactoring`,
  `golang-continuous-integration`, `golang-data-structures`, `golang-dependency-management`,
  `golang-popular-libraries`, `golang-pkg-go-dev`, `golang-stay-updated`.

### Cambios hechos al copiarlas

1. Se quitó la línea `allowed-tools` del frontmatter de cada `SKILL.md`. Esa línea aprobaba de
   antemano `go`, `git`, `golangci-lint` y el lanzamiento de sub-agentes mientras la skill estaba
   activa, salteando los permisos del usuario.
2. No se copiaron las carpetas `evals/`, que solo sirven para evaluar las skills.
3. El contenido de las skills no se modificó. Las decisiones de este proyecto que contradicen sus
   recomendaciones por defecto están en `.claude/CLAUDE.md` § "Precedencia sobre las skills" y
   mandan sobre ellas.

### Revisión de seguridad hecha

- Lectura completa de los `SKILL.md` de `golang-concurrency`, `golang-error-handling`,
  `golang-project-layout` y `golang-design-patterns`.
- En **todos** los archivos copiados, búsqueda automática de: `curl` o `wget` enviados a una shell,
  `rm -rf`, decodificación base64, acceso a `~/.ssh` o credenciales, instrucciones de ignorar
  instrucciones previas o de tocar `settings.json` y `.claude/`. Sin coincidencias peligrosas. La
  única coincidencia de `rm -rf` es `rm -rf bin/` en la plantilla de Makefile.
- Dominios enlazados: github.com, go.dev, pkg.go.dev, uber-go.github.io, samber.github.io,
  prometheus.io, grafana.com, owasp.org, 12factor.net y ejemplos con `localhost`.
- El resto de los `SKILL.md` y las carpetas `references/` **no** se leyeron línea por línea; solo
  pasaron por la búsqueda automática.

### Cómo actualizar

Volver a clonar el repositorio, repetir la selección, quitar `allowed-tools`, repetir la búsqueda
automática y actualizar el commit en este archivo.

## Skills propias del proyecto

- `go-resiliencia`: valores y patrones de resiliencia de esta API (servidor, Redis, ETL).
- `agro-calculos`: reglas para implementar y probar las fórmulas científicas.
