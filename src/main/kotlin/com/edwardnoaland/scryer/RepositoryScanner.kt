package com.edwardnoaland.scryer

import java.nio.file.Files
import java.nio.file.Path
import java.util.Properties
import kotlin.io.path.isDirectory
import kotlin.io.path.isRegularFile

data class BuildFacts(val tool: String, val version: String?, val definition: String)
data class RepositoryFacts(
    val root: Path,
    val builds: List<BuildFacts>,
    val dependencies: List<DependencyDeclaration> = emptyList(),
    val plugins: List<PluginDeclaration> = emptyList(),
    val notes: List<String> = emptyList(),
)

class ScanException(message: String) : RuntimeException(message)

class RepositoryScanner {
    fun scan(path: Path): RepositoryFacts {
        val root = path.toAbsolutePath().normalize()
        if (!root.isDirectory()) throw ScanException("Not a directory: $root")
        val builds = mutableListOf<BuildFacts>()
        for (file in listOf("build.gradle", "build.gradle.kts")) {
            if (root.resolve(file).isRegularFile()) {
                builds += BuildFacts("Gradle", wrapperVersion(root.resolve("gradle/wrapper/gradle-wrapper.properties"),
                    Regex("gradle-(.+)-(?:bin|all)\\.zip")), file)
            }
        }
        if (root.resolve("pom.xml").isRegularFile()) {
            builds += BuildFacts("Maven", wrapperVersion(root.resolve(".mvn/wrapper/maven-wrapper.properties"),
                Regex("apache-maven-(.+)-bin\\.(?:zip|tar\\.gz)")), "pom.xml")
        }
        if (builds.isEmpty()) throw ScanException("No build.gradle, build.gradle.kts or pom.xml found in $root")
        val declarations = builds.map { build ->
            if (build.tool == "Gradle") GradleDeclarations.read(root, build.definition) else MavenDeclarations.read(root)
        }
        return RepositoryFacts(root, builds, declarations.flatMap { it.dependencies },
            declarations.flatMap { it.plugins }, declarations.flatMap { it.notes })
    }

    private fun wrapperVersion(file: Path, pattern: Regex): String? {
        if (!file.isRegularFile()) return null
        val properties = Properties()
        Files.newInputStream(file).use(properties::load)
        val url = properties.getProperty("distributionUrl") ?: return null
        val filename = url.substringBefore('?').substringBefore('#').substringAfterLast('/')
        return pattern.matchEntire(filename)?.groupValues?.get(1)
    }
}
