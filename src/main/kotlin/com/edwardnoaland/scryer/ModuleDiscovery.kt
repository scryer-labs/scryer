package com.edwardnoaland.scryer

import java.nio.file.Path
import javax.xml.XMLConstants
import javax.xml.parsers.DocumentBuilderFactory
import org.w3c.dom.Element
import org.xml.sax.SAXParseException
import org.xml.sax.helpers.DefaultHandler
import kotlin.io.path.*

internal fun readPom(file: Path): Element {
    val factory = DocumentBuilderFactory.newInstance().apply {
        isNamespaceAware = true
        setFeature("http://apache.org/xml/features/disallow-doctype-decl", true)
        setAttribute(XMLConstants.ACCESS_EXTERNAL_DTD, "")
        setAttribute(XMLConstants.ACCESS_EXTERNAL_SCHEMA, "")
    }
    val builder = factory.newDocumentBuilder().apply {
        setErrorHandler(object : DefaultHandler() {
            override fun error(e: SAXParseException) { throw e }
            override fun fatalError(e: SAXParseException) { throw e }
        })
    }
    return try { file.inputStream().use { builder.parse(it).documentElement } }
    catch (e: Exception) { throw ScanException("Cannot read $file: ${e.message}") }
}
internal fun Element.children(name: String): List<Element> = (0 until childNodes.length)
    .mapNotNull { childNodes.item(it) as? Element }.filter { it.localName == name }
internal fun Element.child(name: String): Element? = children(name).firstOrNull()
internal fun Element.text(name: String): String? = child(name)?.textContent?.trim()?.takeIf { it.isNotEmpty() }

internal fun discoverModules(root: Path, builds: List<BuildFacts>): Pair<List<ModuleFacts>, List<String>> {
    val modules = mutableListOf<ModuleFacts>()
    val notes = mutableListOf<String>()
    val visited = mutableSetOf<Path>()
    fun visit(directory: Path, id: String) {
        if (!directory.isDirectory()) { notes += "Module directory missing: $id"; return }
        val real = directory.toRealPath()
        if (!real.startsWith(root.toRealPath())) { notes += "External module not inspected: $id"; return }
        if (!visited.add(real)) return
        val definitions = listOf("build.gradle", "build.gradle.kts", "pom.xml").filter { directory.resolve(it).isRegularFile() }
        for (definition in definitions) modules += ModuleFacts(id, root.relativize(directory).toString().ifEmpty { "." }, definition)
        if (definitions.isEmpty() && id == ":" && builds.any { it.tool == "Gradle" })
            modules += ModuleFacts(id, ".", builds.first { it.tool == "Gradle" }.definition)
        if (directory.resolve("pom.xml").isRegularFile()) {
            val pom = readPom(directory.resolve("pom.xml"))
            val properties = pom.child("properties")?.let { element ->
                (0 until element.childNodes.length).mapNotNull { element.childNodes.item(it) as? Element }
                    .associate { it.localName to it.textContent.trim() }
            }.orEmpty()
            pom.child("modules")?.children("module")?.forEach {
                val path = expandProperties(it.textContent.trim(), properties)
                if ('$' in path) notes += "Unresolved Maven module: $path"
                else visit(directory.resolve(path).normalize(), if (id == ":") path else "$id/$path")
            }
        }
    }
    visit(root, ":")
    for (settings in listOf("settings.gradle", "settings.gradle.kts").map(root::resolve).filter { it.isRegularFile() }) {
        val content = settings.readText().replace(Regex("(?m)//.*$"), "").replace(Regex("(?s)/\\*.*?\\*/"), "")
        val includes = Regex("\\binclude\\s*(?:\\(([^)]*)\\)|([^\\n;]+))").findAll(content)
        for (include in includes) {
            val args = include.groupValues[1].ifEmpty { include.groupValues[2] }
            for (match in Regex("['\"]([^'\"]+)['\"]").findAll(args)) {
                val id = ":" + match.groupValues[1].trimStart(':')
                val mapping = Regex("project\\s*\\(\\s*['\"]${Regex.escape(id)}['\"]\\s*\\)\\.projectDir\\s*=\\s*file\\s*\\(\\s*['\"]([^'\"]+)['\"]\\s*\\)").find(content)
                visit(root.resolve(mapping?.groupValues?.get(1) ?: id.trimStart(':').replace(':', '/')).normalize(), id)
            }
            if (!Regex("['\"]").containsMatchIn(args)) notes += "Computed Gradle module include not resolved: $args"
        }
        if (Regex("\\bincludeBuild|\\bincludeFlat").containsMatchIn(content)) notes += "Composite/includeFlat build topology is not statically expanded."
    }
    return modules to notes
}
