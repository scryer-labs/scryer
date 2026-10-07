package com.edwardnoaland.scryer

import org.w3c.dom.Element
import org.xml.sax.SAXParseException
import org.xml.sax.helpers.DefaultHandler
import java.nio.file.Path
import javax.xml.XMLConstants
import javax.xml.parsers.DocumentBuilderFactory

/** Reads local POM declarations only; no Maven execution, parent/BOM download or profile activation. */
internal object MavenDeclarations {
    fun read(root: Path): Declarations {
        val factory = DocumentBuilderFactory.newInstance().apply {
            isNamespaceAware = true
            setFeature("http://apache.org/xml/features/disallow-doctype-decl", true)
            setFeature("http://xml.org/sax/features/external-general-entities", false)
            setFeature("http://xml.org/sax/features/external-parameter-entities", false)
            setAttribute(XMLConstants.ACCESS_EXTERNAL_DTD, "")
            setAttribute(XMLConstants.ACCESS_EXTERNAL_SCHEMA, "")
        }
        val builder = factory.newDocumentBuilder().apply {
            setErrorHandler(object : DefaultHandler() {
                override fun error(e: SAXParseException) { throw e }
                override fun fatalError(e: SAXParseException) { throw e }
            })
        }
        val project = try {
            root.resolve("pom.xml").toFile().inputStream().use { builder.parse(it).documentElement }
        } catch (e: Exception) {
            throw ScanException("Cannot read pom.xml: ${e.message}")
        }
        if (project.localName != "project") throw ScanException("pom.xml root must be <project>")
        val properties = project.child("properties")?.elements()?.associate { it.localName to it.textContent.trim() }
            .orEmpty().toMutableMap()
        val parent = project.child("parent")
        for (key in listOf("groupId", "artifactId", "version")) {
            (project.text(key) ?: if (key != "artifactId") parent?.text(key) else null)?.let {
                properties["project.$key"] = it
                properties["pom.$key"] = it
            }
        }
        fun expand(value: String?) = value?.let { expandProperties(it, properties) }
        fun dependencies(container: Element?, kind: String): List<DependencyDeclaration> =
            container?.children("dependency")?.map { dep ->
                val group = expand(dep.text("groupId")) ?: "<missing groupId>"
                val name = expand(dep.text("artifactId")) ?: "<missing artifactId>"
                DependencyDeclaration(expand(dep.text("scope")) ?: "compile", "$group:$name",
                    expand(dep.text("version")), "pom.xml", kind, declaredVersion = dep.text("version"),
                    rawDeclaration = "${dep.text("groupId")}:${dep.text("artifactId")}:${dep.text("version") ?: "<unspecified>"}")
            }.orEmpty()
        val managedContainer = project.child("dependencyManagement")?.child("dependencies")
        val managed = dependencies(managedContainer, "managed declaration")
        val managedElements = managedContainer?.children("dependency").orEmpty()
        val declaredElements = project.child("dependencies")?.children("dependency").orEmpty()
        val declared = dependencies(project.child("dependencies"), "dependency").mapIndexed { index, dep ->
            // Local dependencyManagement can supply a version without any remote/model execution.
            if (dep.version != null) dep else {
                val element = declaredElements[index]
                val matches = managed.withIndex().filter { (managedIndex, candidate) ->
                    val definition = managedElements[managedIndex]
                    candidate.notation == dep.notation && candidate.configuration != "import" &&
                        (expand(element.text("type")) ?: "jar") == (expand(definition.text("type")) ?: "jar") &&
                        expand(element.text("classifier")) == expand(definition.text("classifier"))
                }
                val match = matches.singleOrNull()?.value
                dep.copy(version = match?.version, declaredVersion = dep.declaredVersion,
                    managementSource = if (match?.version != null) "pom.xml dependencyManagement" else null,
                    kind = if (match?.version != null) "dependency (local management)" else "dependency")
            }
        }
        val plugins = project.child("build")?.child("plugins")?.children("plugin")?.map { plugin ->
            PluginDeclaration("${expand(plugin.text("groupId")) ?: "org.apache.maven.plugins"}:${expand(plugin.text("artifactId"))}",
                expand(plugin.text("version")), "pom.xml")
        }.orEmpty()
        val notes = mutableListOf<String>()
        if (parent != null) notes += "Parent POM: ${expand(parent.text("groupId"))}:${expand(parent.text("artifactId"))}:${expand(parent.text("version"))}; parent inheritance is not resolved."
        if (project.child("modules") != null) notes += "Maven modules detected: this slice inspects root declarations only; module dependencies are not expanded."
        if (project.child("profiles") != null) notes += "Maven profiles detected: profile-specific declarations are not expanded."
        notes += "Maven parent/BOM inheritance, profiles and transitive versions require model resolution; only local declarations are reported."
        return Declarations(declared + managed, plugins, notes)
    }

    private fun Element.elements(): List<Element> = (0 until childNodes.length).mapNotNull { childNodes.item(it) as? Element }
    private fun Element.children(name: String) = elements().filter { it.localName == name }
    private fun Element.child(name: String) = children(name).firstOrNull()
    private fun Element.text(name: String) = child(name)?.textContent?.trim()?.takeIf { it.isNotEmpty() }
}
