package com.edwardnoaland.scryer.cli

import com.edwardnoaland.scryer.cli.output.ScanRenderer
import com.edwardnoaland.scryer.cli.output.renderJson
import com.edwardnoaland.scryer.cli.output.writeMarkdownReport
import com.edwardnoaland.scryer.scan.ScanService
import com.edwardnoaland.scryer.scan.model.RepositoryFacts
import com.edwardnoaland.scryer.scan.resolve.DependencyResolver
import java.io.PrintStream
import java.nio.file.Path

internal const val SCAN_USAGE = "Usage: scryer scan <path> [--dependencies] [--dependency-tree] [--json] [--static] [--color auto|always|never] [-o <OUTPUT_FILE>]"

private data class ScanOptions(
    val path: String,
    val dependencies: Boolean,
    val tree: Boolean,
    val json: Boolean,
    val staticOnly: Boolean,
    val color: String,
    val outputFile: String?,
)

private fun parseScanOptions(args: Array<String>): ScanOptions? {
    if (args.isEmpty() || args[0].startsWith('-')) {
        return null
    }

    var dependencies = false
    var tree = false
    var json = false
    var staticOnly = false
    var color = "auto"
    var outputFile: String? = null
    var index = 1

    while (index < args.size) {
        when (args[index]) {
            "--dependencies" -> dependencies = true
            "--dependency-tree" -> tree = true
            "--json" -> json = true
            "--static" -> staticOnly = true
            "-o", "--output" -> {
                index++
                if (index >= args.size || args[index].isBlank() || args[index].startsWith('-')) {
                    return null
                }
                outputFile = args[index]
            }
            "--color" -> {
                index++
                if (index >= args.size || args[index] !in listOf("auto", "always", "never")) {
                    return null
                }
                color = args[index]
            }
            else -> return null
        }
        index++
    }

    return ScanOptions(args[0], dependencies, tree, json, staticOnly, color, outputFile)
}

internal fun runScanCommand(args: Array<String>, out: PrintStream, err: PrintStream): Int {
    val options = parseScanOptions(args)
    if (options == null) {
        err.println(CLI_USAGE)
        return 2
    }

    return try {
        val resolver = DependencyResolver(progress = { message -> err.println("scryer: $message") })
        val service = ScanService(resolver = resolver)
        val facts = service.scan(Path.of(options.path), staticOnly = options.staticOnly)
        options.outputFile?.let { filename ->
            val destination = Path.of(filename).toAbsolutePath().normalize()
            writeMarkdownReport(facts, destination, includeTree = options.tree)
            err.println("scryer: Markdown report written to $destination")
        }
        renderScan(facts, options, out)
        0
    } catch (exception: Exception) {
        err.println("scryer: ${exception.message ?: exception.javaClass.simpleName}")
        1
    }
}

private fun renderScan(facts: RepositoryFacts, options: ScanOptions, out: PrintStream) {
    if (options.json) {
        renderJson(facts, out)
        return
    }

    val colorEnabled = when (options.color) {
        "always" -> true
        "never" -> false
        else -> System.getenv("NO_COLOR") == null && System.console() != null && out === System.out
    }
    val renderer = ScanRenderer(out, colorEnabled)
    renderer.summary(facts)
    if (options.dependencies) {
        renderer.dependencies(facts)
    }
    if (options.tree) {
        renderer.tree(facts)
    }
}
