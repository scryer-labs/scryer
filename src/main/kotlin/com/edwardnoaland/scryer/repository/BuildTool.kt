package com.edwardnoaland.scryer.repository

/** Explicit selection shared by repository inspection, model collection and test planning. */
enum class BuildTool(val flag: String, val displayName: String, val wrapper: String) {
    MAVEN("maven", "Maven", "mvnw"),
    GRADLE("gradle", "Gradle", "gradlew");

    companion object {
        fun parse(value: String): BuildTool? = entries.firstOrNull { it.flag == value }
    }
}
