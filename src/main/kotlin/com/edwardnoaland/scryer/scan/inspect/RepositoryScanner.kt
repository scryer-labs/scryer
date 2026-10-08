package com.edwardnoaland.scryer.scan.inspect

import com.edwardnoaland.scryer.scan.model.BuildFacts
import com.edwardnoaland.scryer.scan.model.CompileToolingFacts
import com.edwardnoaland.scryer.scan.model.Declarations
import com.edwardnoaland.scryer.scan.model.DependencyDeclaration
import com.edwardnoaland.scryer.scan.model.ModuleFacts
import com.edwardnoaland.scryer.scan.model.ParentPomFact
import com.edwardnoaland.scryer.scan.model.RepositoryFacts
import com.edwardnoaland.scryer.scan.model.ScanException
import com.edwardnoaland.scryer.scan.model.SignalFact
import java.nio.file.Files
import java.nio.file.Path
import java.util.Properties
import kotlin.io.path.isDirectory
import kotlin.io.path.isRegularFile
import kotlin.io.path.readText
import org.w3c.dom.Element

/** Collects local declarations and source signals without executing the target build. */
class RepositoryScanner {
    fun scan(path: Path): RepositoryFacts {
        val root = path.toAbsolutePath().normalize()
        if (!root.isDirectory()) {
            throw ScanException("Not a directory: $root")
        }

        val builds = detectBuilds(root)
        if (builds.isEmpty()) {
            throw ScanException("No build.gradle, build.gradle.kts or pom.xml found in $root")
        }

        val (modules, moduleNotes) = discoverModules(root, builds)
        val declarations = modules.map { module -> readDeclarations(root, module) }
        val dependencies = declarations.flatMap { it.dependencies }
        val plugins = declarations.flatMap { it.plugins }
        val inspection = inspectMetadata(root, modules, dependencies, plugins)
        val declarationNotes = declarations.flatMap { it.notes }
            .filterNot { "root declarations only" in it }

        return RepositoryFacts(
            root = root,
            builds = builds,
            dependencies = dependencies,
            plugins = plugins,
            notes = declarationNotes + moduleNotes,
            modules = modules,
            language = inspection.language,
            sources = inspection.sources,
            testing = inspection.testing,
            signals = inspection.signals,
            verification = inspection.verification,
            repositories = inspection.repositories,
            customBuildFiles = inspection.customBuildFiles,
            frameworks = inspection.frameworks,
            parentPoms = readParentPoms(root, modules),
            compileTooling = inspectCompileTooling(root, modules, dependencies, inspection.signals),
        )
    }

    private fun detectBuilds(root: Path): List<BuildFacts> {
        val builds = mutableListOf<BuildFacts>()
        val gradleDefinitions = listOf("build.gradle", "build.gradle.kts")
            .filter { root.resolve(it).isRegularFile() }

        gradleDefinitions.forEach { definition -> builds += gradleFacts(root, definition) }
        if (gradleDefinitions.isEmpty()) {
            val settings = listOf("settings.gradle", "settings.gradle.kts")
                .firstOrNull { root.resolve(it).isRegularFile() }
            if (settings != null) {
                builds += gradleFacts(root, settings)
            }
        }

        if (root.resolve("pom.xml").isRegularFile()) {
            builds += BuildFacts(
                tool = "Maven",
                version = wrapperVersion(
                    root.resolve(".mvn/wrapper/maven-wrapper.properties"),
                    Regex("apache-maven-(.+)-bin\\.(?:zip|tar\\.gz)"),
                ),
                definition = "pom.xml",
                wrapperPresent = root.resolve("mvnw").isRegularFile(),
            )
        }
        return builds
    }

    private fun gradleFacts(root: Path, definition: String) = BuildFacts(
        tool = "Gradle",
        version = wrapperVersion(
            root.resolve("gradle/wrapper/gradle-wrapper.properties"),
            Regex("gradle-(.+)-(?:bin|all)\\.zip"),
        ),
        definition = definition,
        wrapperPresent = root.resolve("gradlew").isRegularFile(),
    )

    private fun readDeclarations(root: Path, module: ModuleFacts): Declarations {
        val directory = root.resolve(module.directory)
        val declarations = when {
            module.definition == "pom.xml" -> MavenDeclarations.read(directory)
            module.definition.startsWith("build.gradle") -> GradleDeclarations.read(directory, module.definition)
            else -> Declarations()
        }
        val dependencies = declarations.dependencies.map { dependency ->
            val source = if (module.directory == ".") {
                dependency.source
            } else {
                "${module.directory}/${dependency.source}"
            }
            dependency.copy(module = module.id, source = source)
        }
        return declarations.copy(dependencies = dependencies)
    }

    private fun readParentPoms(root: Path, modules: List<ModuleFacts>): List<ParentPomFact> =
        modules.filter { it.definition == "pom.xml" }.mapNotNull { module ->
            val parent = readPom(root.resolve(module.directory).resolve("pom.xml")).child("parent")
                ?: return@mapNotNull null
            ParentPomFact(
                module = module.id,
                group = parent.text("groupId"),
                artifact = parent.text("artifactId"),
                declaredVersion = parent.text("version"),
            )
        }

    private fun inspectCompileTooling(
        root: Path,
        modules: List<ModuleFacts>,
        dependencies: List<DependencyDeclaration>,
        signals: List<SignalFact>,
    ): CompileToolingFacts {
        val gradleProcessors = dependencies.filter {
            "annotationProcessor" in it.configuration || "kapt" in it.configuration
        }
        val mavenProcessors = modules.filter { it.definition == "pom.xml" }
            .flatMap { readMavenProcessors(root, it) }
        val buildText = modules.map { root.resolve(it.directory).resolve(it.definition) }
            .filter { it.isRegularFile() }
            .joinToString("\n") { it.readText() }
            .lowercase()
        val generators = listOf("mapstruct", "querydsl", "protobuf", "openapi", "lombok")
            .filter { it in buildText }
        val usesLombok = dependencies.any { "lombok" in it.notation } ||
            signals.any { it.name == "Lombok" && it.locations.isNotEmpty() }

        return CompileToolingFacts(gradleProcessors + mavenProcessors, generators, usesLombok)
    }

    private fun readMavenProcessors(root: Path, module: ModuleFacts): List<DependencyDeclaration> {
        val pom = readPom(root.resolve(module.directory).resolve("pom.xml"))
        val paths = pom.getElementsByTagNameNS("*", "annotationProcessorPaths")
        return (0 until paths.length).flatMap { index ->
            val element = paths.item(index) as Element
            element.children("path").map { processor ->
                DependencyDeclaration(
                    configuration = "annotationProcessor",
                    notation = "${processor.text("groupId")}:${processor.text("artifactId")}",
                    version = processor.text("version"),
                    source = "${module.directory}/pom.xml",
                    module = module.id,
                )
            }
        }
    }

    private fun wrapperVersion(file: Path, pattern: Regex): String? {
        if (!file.isRegularFile()) {
            return null
        }
        val properties = Properties()
        Files.newInputStream(file).use(properties::load)
        val url = properties.getProperty("distributionUrl") ?: return null
        val filename = url.substringBefore('?').substringBefore('#').substringAfterLast('/')
        return pattern.matchEntire(filename)?.groupValues?.get(1)
    }
}
