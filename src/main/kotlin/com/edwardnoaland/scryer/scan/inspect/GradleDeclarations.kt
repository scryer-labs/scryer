package com.edwardnoaland.scryer.scan.inspect

import com.edwardnoaland.scryer.scan.model.Declarations
import com.edwardnoaland.scryer.scan.model.DependencyDeclaration
import com.edwardnoaland.scryer.scan.model.PluginDeclaration
import java.nio.file.Path
import java.util.Properties
import kotlin.io.path.isRegularFile
import kotlin.io.path.readText

/** Best-effort literal inspection, not a Groovy/Kotlin interpreter or Gradle model. */
internal object GradleDeclarations {
    fun read(root: Path, filename: String): Declarations {
        val text = stripComments(root.resolve(filename).readText())
        val properties = mutableMapOf<String, String>()
        val propertiesFile = root.resolve("gradle.properties")
        if (propertiesFile.isRegularFile()) {
            val loaded = Properties()
            propertiesFile.toFile().inputStream().use(loaded::load)
            loaded.stringPropertyNames().forEach { properties[it] = loaded.getProperty(it) }
        }
        Regex("(?:\\b(?:val|var|def)\\s+)?([A-Za-z_]\\w*)\\s*=\\s*(['\"])(.*?)\\2")
            .findAll(text).forEach { properties[it.groupValues[1]] = it.groupValues[3] }
        val notes = mutableListOf<String>()
        val dependencies = mutableListOf<DependencyDeclaration>()
        for (body in blocks(text, "dependencies")) {
            for (statement in statements(body)) {
                val config = Regex("^([A-Za-z_]\\w*)\\b").find(statement)?.groupValues?.get(1)
                if (config == null) {
                    notes += "$filename: unsupported dependency declaration: ${statement.take(120)}"
                    continue
                }
                val argument = statement.substring(config.length).trim()
                val closureStart = maskStrings(argument).indexOf('{')
                val leadingArgument = if (closureStart < 0) argument else argument.substring(0, closureStart)
                val mapValues = Regex("\\b(group|name|version)\\s*:\\s*(['\"])(.*?)\\2")
                    .findAll(leadingArgument).associate { it.groupValues[1] to expandProperties(it.groupValues[3], properties) }
                if (mapValues.containsKey("group") && mapValues.containsKey("name")) {
                    val versionExpression = Regex("\\bversion\\s*:\\s*([^\\s,)]+)")
                        .find(leadingArgument)?.groupValues?.get(1)
                    val version = mapValues["version"] ?: versionExpression?.let {
                        expandProperties("$" + "{" + it + "}", properties)
                    }
                    dependencies += DependencyDeclaration(config, "${mapValues["group"]}:${mapValues["name"]}",
                        version, filename,
                        declaredVersion = Regex("\\bversion\\s*:\\s*(['\"])(.*?)\\1").find(leadingArgument)?.groupValues?.get(2) ?: versionExpression,
                        rawDeclaration = statement)
                    continue
                }
                // Only a leading literal or supported platform(literal); don't mine nested closures.
                val literal = Regex("^\\(?\\s*(?:(platform|enforcedPlatform)\\s*\\(\\s*)?(['\"])(.*?)\\2")
                    .find(argument)
                val coordinate = literal?.groupValues?.get(3)?.let { expandProperties(it, properties) }
                val parts = coordinate?.split(':')
                if (parts != null && parts.size in 2..3 && parts.take(2).all { it.isNotBlank() }) {
                    dependencies += DependencyDeclaration(config, parts.take(2).joinToString(":"),
                        parts.getOrNull(2), filename, if (literal.groupValues[1].isEmpty()) "dependency" else "platform",
                        declaredVersion = literal.groupValues[3].split(':').getOrNull(2), rawDeclaration = statement)
                } else {
                    notes += "$filename: unsupported dependency declaration: ${statement.take(120)}"
                }
            }
        }
        val plugins = blocks(text, "plugins").flatMap { body ->
            Regex("\\bid\\s*\\(?\\s*(['\"])(.*?)\\1\\s*\\)?\\s*version\\s*\\(?\\s*(['\"])(.*?)\\3")
                .findAll(body).map { PluginDeclaration(it.groupValues[2], expandProperties(it.groupValues[4], properties), filename) }.toList() +
                Regex("\\bkotlin\\s*\\(\\s*\"([^\"]+)\"\\s*\\)\\s*version\\s*\"([^\"]+)\"")
                    .findAll(body).map { PluginDeclaration("org.jetbrains.kotlin.${it.groupValues[1]}",
                        expandProperties(it.groupValues[2], properties), filename) }.toList()
        }
        if (root.resolve("settings.gradle").isRegularFile() || root.resolve("settings.gradle.kts").isRegularFile()) {
            val settings = listOf("settings.gradle", "settings.gradle.kts").filter { root.resolve(it).isRegularFile() }
                .joinToString("\n") { stripComments(root.resolve(it).readText()) }
            if (Regex("\\binclude(?:Flat)?\\b").containsMatchIn(settings)) {
                notes += "Gradle includes detected: this slice inspects root declarations only; module dependencies are not expanded."
            }
        }
        return Declarations(dependencies.distinct(), plugins.distinct(), notes.distinct())
    }

    private fun stripComments(text: String): String {
        val out = StringBuilder()
        var i = 0
        var quote: Char? = null
        while (i < text.length) {
            val c = text[i]
            when {
                quote != null -> {
                    out.append(c)
                    if (c == '\\' && i + 1 < text.length) out.append(text[++i])
                    else if (c == quote) quote = null
                    i++
                }
                c == '\'' || c == '"' -> { quote = c; out.append(c); i++ }
                text.startsWith("//", i) -> {
                    while (i < text.length && text[i] != '\n') { out.append(' '); i++ }
                }
                text.startsWith("/*", i) -> {
                    out.append("  "); i += 2
                    while (i < text.length && !text.startsWith("*/", i)) {
                        out.append(if (text[i] == '\n') '\n' else ' '); i++
                    }
                    if (i < text.length) { out.append("  "); i += 2 }
                }
                else -> { out.append(c); i++ }
            }
        }
        return out.toString()
    }

    private fun maskStrings(text: String): String {
        val out = text.toCharArray()
        var quote: Char? = null
        var escaped = false
        for (i in text.indices) {
            val c = text[i]
            if (quote != null) {
                if (c != '\n') out[i] = ' '
                if (escaped) escaped = false
                else if (c == '\\') escaped = true
                else if (c == quote) quote = null
            } else if (c == '\'' || c == '"') { quote = c; out[i] = ' ' }
        }
        return String(out)
    }

    private fun blocks(text: String, name: String): List<String> {
        val masked = maskStrings(text)
        return Regex("\\b$name\\s*\\{").findAll(masked).mapNotNull { match ->
            val start = match.range.last + 1
            var depth = 1
            var end = start
            while (end < masked.length && depth > 0) {
                if (masked[end] == '{') depth++
                if (masked[end] == '}') depth--
                end++
            }
            if (depth == 0) text.substring(start, end - 1) else null
        }.toList()
    }

    private fun statements(body: String): List<String> {
        val masked = maskStrings(body)
        val result = mutableListOf<String>()
        var depth = 0
        var start = 0
        for (i in masked.indices) {
            when (masked[i]) { '(', '{', '[' -> depth++; ')', '}', ']' -> depth-- }
            if (depth == 0 && (masked[i] == ';' || masked[i] == '\n') &&
                !body.substring(start, i).trimEnd().endsWith(',')) {
                body.substring(start, i).trim().takeIf { it.isNotEmpty() }?.let(result::add)
                start = i + 1
            }
        }
        body.substring(start).trim().takeIf { it.isNotEmpty() }?.let(result::add)
        return result
    }
}
