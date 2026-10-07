package com.edwardnoaland.scryer

internal fun withResolution(facts: RepositoryFacts, resolution: ResolutionFacts): RepositoryFacts {
    val modules = resolution.configurations.flatMap { it.nodes }.filter { it.group != null }
    val frameworks = facts.frameworks.toMutableSet()
    val frameworkMatchers = mapOf("Spring Boot" to "org.springframework.boot", "Spring Framework" to "org.springframework",
        "Spring Security" to "org.springframework.security", "Hibernate/JPA" to "org.hibernate", "Tomcat" to "org.apache.tomcat",
        "Jetty" to "org.eclipse.jetty")
    for ((name, group) in frameworkMatchers) if (modules.any { it.group!!.startsWith(group) }) frameworks += name
    val testFrameworks = facts.testing.frameworks.toMutableSet()
    if (modules.any { it.group == "junit" && it.artifact == "junit" }) testFrameworks += "JUnit 4"
    if (modules.any { it.group == "org.junit.jupiter" }) testFrameworks += "JUnit 5"
    if (modules.any { it.group == "org.testng" }) testFrameworks += "TestNG"
    fun library(prefix: String) = modules.filter { it.group!!.startsWith(prefix) }
        .map { "${it.group}:${it.artifact}:${it.version}" }.distinct()
    fun version(value: String) = value.removePrefix("1.")
    val dependencies = facts.dependencies.map { dep -> dep.copy(resolvedVersions = resolution.configurations.filter { it.module == dep.module }
        .flatMap { it.nodes }.filter { it.group == dep.group && it.artifact == dep.artifact }.mapNotNull { it.version }.distinct().sorted()) }
    return facts.copy(resolution = resolution, dependencies = dependencies,
        language = facts.language.copy(
            sourceVersions = (facts.language.sourceVersions + resolution.projects.mapNotNull { it.sourceVersion?.let(::version) }).distinct(),
            targetVersions = (facts.language.targetVersions + resolution.projects.mapNotNull { it.targetVersion?.let(::version) }).distinct(),
            toolchainVersions = (facts.language.toolchainVersions + resolution.projects.mapNotNull { it.toolchainVersion }).distinct()),
        frameworks = frameworks.toList(),
        testing = facts.testing.copy(frameworks = testFrameworks.toList(), mockLibraries = (facts.testing.mockLibraries + library("org.mockito")).distinct(),
            assertionLibraries = (facts.testing.assertionLibraries + library("org.assertj") + library("org.hamcrest")).distinct()))
}
