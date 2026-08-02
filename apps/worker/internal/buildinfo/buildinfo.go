// Package buildinfo dá ao worker o mesmo esquema de versionamento já
// aplicado a web/iOS/local-agent/api (skill ildemar_app-versioning): fonte
// única de versão (VERSION na raiz do módulo) e commit injetado no build,
// nunca hardcoded.
package buildinfo

// Version e GitCommit são sobrescritos em tempo de build via ldflags
// (-X egger/worker/internal/buildinfo.Version=..., ver apps/worker/Dockerfile).
var (
	Version   = "dev"
	GitCommit = "dev"
)
