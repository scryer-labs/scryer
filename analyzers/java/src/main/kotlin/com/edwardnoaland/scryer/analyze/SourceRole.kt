package com.edwardnoaland.scryer.analyze

/** Conventional source roots only; unfamiliar layouts stay unknown. */
enum class SourceRole { PRODUCTION, TEST, UNKNOWN }
internal fun sourceRole(path: String): SourceRole {
    val normalized = "/${path.replace('\\', '/')}"
    return when {
        listOf("/src/test/", "/src/integrationTest/", "/src/integration-test/", "/src/testFixtures/").any { it in normalized } -> SourceRole.TEST
        "/src/main/" in normalized -> SourceRole.PRODUCTION
        else -> SourceRole.UNKNOWN
    }
}
