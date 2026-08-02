// Package buildinfo dá ao backend Go o mesmo esquema de versionamento já
// aplicado a web/iOS/local-agent (skill ildemar_app-versioning): fonte
// única de versão (VERSION na raiz do módulo) e commit injetado no build,
// nunca hardcoded — os fallbacks abaixo só valem pra builds locais
// (`go run`/`go build` sem ldflags).
package buildinfo

// Version e GitCommit são sobrescritos em tempo de build via ldflags
// (-X egger/api/internal/buildinfo.Version=..., ver apps/api/Dockerfile).
var (
	Version   = "dev"
	GitCommit = "dev"
)
