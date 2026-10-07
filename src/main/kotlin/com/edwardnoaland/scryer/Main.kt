package com.edwardnoaland.scryer

import java.io.PrintStream
import java.nio.file.Path
import kotlin.system.exitProcess

private const val USAGE = "Usage: scryer scan <path> [--dependencies] [--dependency-tree] [--json] [--static] [--color auto|always|never]"
private data class Options(val path: String, val dependencies: Boolean, val tree: Boolean, val json: Boolean, val static: Boolean, val color: String)

private fun parseOptions(args: Array<String>): Options? {
    if (args.size < 2 || args[0] != "scan" || args[1].startsWith('-')) return null
    var dependencies = false; var tree = false; var json = false; var static = false; var color = "auto"
    var i = 2
    while (i < args.size) {
        when (args[i]) {
            "--dependencies" -> dependencies = true
            "--dependency-tree" -> tree = true
            "--json" -> json = true
            "--static" -> static = true
            "--color" -> { if (++i >= args.size || args[i] !in listOf("auto", "always", "never")) return null; color = args[i] }
            else -> return null
        }
        i++
    }
    return Options(args[1], dependencies, tree, json, static, color)
}

fun runCli(args: Array<String>, out: PrintStream, err: PrintStream): Int {
    if (args.contentEquals(arrayOf("--help")) || args.contentEquals(arrayOf("-h"))) { out.println(USAGE); return 0 }
    val options = parseOptions(args) ?: run { err.println(USAGE); return 2 }
    return try {
        var facts = RepositoryScanner().scan(Path.of(options.path))
        if (!options.static) {
            val resolution = DependencyResolver(progress = { err.println("scryer: $it") }).resolve(facts)
            facts = withResolution(facts, resolution)
        }
        if (options.json) {
            val json = jsonMapper.valueToTree<com.fasterxml.jackson.databind.node.ObjectNode>(facts)
            json.put("root", facts.root.toString())
            out.println(jsonMapper.writerWithDefaultPrettyPrinter().writeValueAsString(json))
        } else {
            val color = options.color == "always" || (options.color == "auto" && System.getenv("NO_COLOR") == null && System.console() != null && out === System.out)
            val renderer = ScanRenderer(out, color)
            renderer.summary(facts)
            if (options.dependencies) renderer.dependencies(facts)
            if (options.tree) renderer.tree(facts)
        }
        0
    } catch (e: Exception) {
        err.println("scryer: ${e.message ?: e.javaClass.simpleName}")
        1
    }
}

fun main(args: Array<String>) { exitProcess(runCli(args, System.out, System.err)) }
