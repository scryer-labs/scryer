package com.edwardnoaland.scryer.scan.model

/** Declared facts, deliberately distinct from a build tool's resolved dependency graph. */
data class DependencyDeclaration(
    val configuration: String,
    val notation: String,
    val version: String?,
    val source: String,
    val kind: String = "dependency",
    val module: String = ":",
    val declaredVersion: String? = version,
    val managementSource: String? = null,
    val rawDeclaration: String? = null,
    val resolvedVersions: List<String> = emptyList(),
    val direct: Boolean = kind != "managed declaration",
    val type: String? = null,
    val classifier: String? = null,
    val optional: Boolean? = null,
) {
    val group: String get() = notation.substringBefore(':')
    val artifact: String get() = notation.substringAfter(':', "")
    @get:com.fasterxml.jackson.annotation.JsonIgnore
    val versionDisplay: String get() = when {
        version == null -> "unspecified (possibly managed; not resolved)"
        '$' in version -> "$version (unresolved expression)"
        else -> version
    }
}

data class PluginDeclaration(val id: String, val version: String?, val source: String)
data class Declarations(
    val dependencies: List<DependencyDeclaration> = emptyList(),
    val plugins: List<PluginDeclaration> = emptyList(),
    val notes: List<String> = emptyList(),
)

