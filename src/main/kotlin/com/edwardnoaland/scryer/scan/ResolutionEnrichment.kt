package com.edwardnoaland.scryer.scan

import com.edwardnoaland.scryer.scan.model.RepositoryFacts
import com.edwardnoaland.scryer.scan.model.ResolutionFacts

/** Adds resolved observations while retaining the original declarations and their provenance. */
internal fun withResolution(facts: RepositoryFacts, resolution: ResolutionFacts): RepositoryFacts {
    val components = resolution.configurations.flatMap { it.nodes }.filter { it.group != null }
    val frameworks = facts.frameworks.toMutableSet()
    val frameworkMatchers = mapOf(
        "Spring Boot" to "org.springframework.boot",
        "Spring Framework" to "org.springframework",
        "Spring Security" to "org.springframework.security",
        "Hibernate/JPA" to "org.hibernate",
        "Tomcat" to "org.apache.tomcat",
        "Jetty" to "org.eclipse.jetty",
    )
    frameworkMatchers.forEach { (name, group) ->
        if (components.any { it.group?.startsWith(group) == true }) {
            frameworks += name
        }
    }

    val testFrameworks = facts.testing.frameworks.toMutableSet()
    if (components.any { it.group == "junit" && it.artifact == "junit" }) {
        testFrameworks += "JUnit 4"
    }
    val jupiterVersions = components.filter { it.group == "org.junit.jupiter" }
        .mapNotNull { it.version?.substringBefore('.')?.toIntOrNull() }
        .distinct()
        .sorted()
    if (jupiterVersions.isNotEmpty()) {
        testFrameworks.remove("JUnit Jupiter")
        jupiterVersions.forEach { major -> testFrameworks += "JUnit Jupiter $major" }
    } else if (components.any { it.group == "org.junit.jupiter" }) {
        testFrameworks += "JUnit Jupiter"
    }
    if (components.any { it.group == "org.testng" }) {
        testFrameworks += "TestNG"
    }

    fun libraries(groupPrefix: String): List<String> = components
        .filter { it.group?.startsWith(groupPrefix) == true }
        .map { "${it.group}:${it.artifact}:${it.version}" }
        .distinct()

    val dependencies = facts.dependencies.map { declaration ->
        val selectedVersions = resolution.configurations.filter { it.module == declaration.module }
            .flatMap { it.nodes }
            .filter { it.group == declaration.group && it.artifact == declaration.artifact }
            .mapNotNull { it.version }
            .distinct()
            .sorted()
        declaration.copy(resolvedVersions = selectedVersions)
    }
    val sourceVersions = resolution.projects.mapNotNull { it.sourceVersion?.removePrefix("1.") }
    val targetVersions = resolution.projects.mapNotNull { it.targetVersion?.removePrefix("1.") }
    val toolchainVersions = resolution.projects.mapNotNull { it.toolchainVersion }

    return facts.copy(
        resolution = resolution,
        dependencies = dependencies,
        language = facts.language.copy(
            sourceVersions = (facts.language.sourceVersions + sourceVersions).distinct(),
            targetVersions = (facts.language.targetVersions + targetVersions).distinct(),
            toolchainVersions = (facts.language.toolchainVersions + toolchainVersions).distinct(),
        ),
        frameworks = frameworks.toList(),
        testing = facts.testing.copy(
            frameworks = testFrameworks.toList(),
            mockLibraries = (facts.testing.mockLibraries + libraries("org.mockito")).distinct(),
            assertionLibraries = (facts.testing.assertionLibraries + libraries("org.assertj") + libraries("org.hamcrest")).distinct(),
        ),
    )
}
