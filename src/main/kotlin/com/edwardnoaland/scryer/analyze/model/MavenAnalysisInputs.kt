package com.edwardnoaland.scryer.analyze.model

import com.edwardnoaland.scryer.analyze.SourceRole
import com.edwardnoaland.scryer.analyze.execute.TargetTestRunner
import com.edwardnoaland.scryer.analyze.execute.TestRunPlan
import com.edwardnoaland.scryer.analyze.execute.TestRunStatus
import org.w3c.dom.Element
import java.nio.file.Files
import java.nio.file.Path
import javax.xml.XMLConstants
import javax.xml.parsers.DocumentBuilderFactory

internal class MavenAnalysisInputs(
    private val runner: TargetTestRunner,
    private val plan: TestRunPlan,
    private val run: Path,
    private val timeout: Long,
    private val sha: String,
) {
    fun read(output: Path, root: Path): AnalysisInputs {
        val factory = DocumentBuilderFactory.newInstance()
        factory.setFeature("http://apache.org/xml/features/disallow-doctype-decl", true)
        factory.setAttribute(XMLConstants.ACCESS_EXTERNAL_DTD, "")
        factory.setAttribute(XMLConstants.ACCESS_EXTERNAL_SCHEMA, "")
        val document = factory.newDocumentBuilder().parse(output.toFile()).documentElement
        val projects = if (document.tagName == "project") listOf(document) else document.children("project")
        // Effective POM does not reliably expose basedir; follow the reactor's module declarations.
        val directories = linkedSetOf(root)
        fun discover(directory: Path) {
            val pom = directory.resolve("pom.xml")
            if (!Files.isRegularFile(pom)) return
            val local = factory.newDocumentBuilder().parse(pom.toFile()).documentElement
            local.child("modules")?.children("module").orEmpty().forEach {
                val child = directory.resolve(it.textContent.trim()).normalize()
                if (child.startsWith(root) && directories.add(child)) discover(child)
            }
        }
        discover(root)
        val byArtifact = directories.groupBy { directory ->
            val local = factory.newDocumentBuilder().parse(directory.resolve("pom.xml").toFile()).documentElement
            local.text("artifactId")
        }
        val coordinates = byArtifact.filterValues { it.size == 1 }.mapValues { it.value.single() }
        val modules = projects.mapIndexedNotNull { index, project ->
            val artifact = project.text("artifactId")
            val directory = coordinates[artifact] ?: return@mapIndexedNotNull null
            val build = project.child("build")
            fun source(name: String, fallback: String) = directory.resolve(build?.text(name)?.takeIf { it.isNotBlank() } ?: fallback).normalize()
            val classpathFile = run.resolve("classpath-$index.txt")
            val command = plan.command.dropLast(2) + listOf("-N", "-f", directory.resolve("pom.xml").toString(),
                "org.apache.maven.plugins:maven-dependency-plugin:3.8.1:build-classpath", "-DincludeScope=test", "-Dmdep.outputFile=$classpathFile")
            val result = runner.execute(root, plan.copy(command = command), run.resolve("module-$index.log"), sha, timeout, "module classpath")
            val classpath = if (result.status == TestRunStatus.SUCCEEDED && Files.isRegularFile(classpathFile))
                Files.readString(classpathFile).trim().split(java.io.File.pathSeparator).filter { it.isNotBlank() }.map(Path::of) else emptyList()
            val dependencies = project.child("dependencies")?.children("dependency").orEmpty().mapNotNull {
                coordinates[it.text("artifactId")]?.let { path -> root.relativize(path).toString().ifEmpty { ":" } }
            }
            AnalysisModule(root.relativize(directory).toString().ifEmpty { ":" }, directory,
                listOf(AnalysisSourceRoot(source("sourceDirectory", "src/main/java"), SourceRole.PRODUCTION),
                    AnalysisSourceRoot(source("testSourceDirectory", "src/test/java"), SourceRole.TEST)), classpath, dependencies,
                if (result.status == TestRunStatus.SUCCEEDED) emptyList() else listOf("Dependency classpath unavailable (${result.status}); log: ${result.log}"))
        }
        return AnalysisInputs(modules, listOf("Maven effective-POM source roots and test-scope dependency classpath. Reactor dependencies use source attribution; generated roots, duplicate artifact IDs and profile-only modules may remain unavailable.") + if (byArtifact.any { it.value.size > 1 }) listOf("Duplicate reactor artifact IDs excluded from module association.") else emptyList())
    }

    private fun Element.children(name: String): List<Element> = (0 until childNodes.length).mapNotNull { childNodes.item(it) as? Element }.filter { it.tagName == name }
    private fun Element.child(name: String) = children(name).firstOrNull()
    private fun Element.text(name: String) = child(name)?.textContent?.trim().orEmpty()
}
