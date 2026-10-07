package com.edwardnoaland.scryer

import java.io.PrintStream
import java.nio.file.Path
import kotlin.system.exitProcess

private const val USAGE = "Usage: scryer scan <path>"

fun runCli(args: Array<String>, out: PrintStream, err: PrintStream): Int {
    if (args.contentEquals(arrayOf("--help")) || args.contentEquals(arrayOf("-h"))) {
        out.println(USAGE)
        return 0
    }
    if (args.size != 2 || args[0] != "scan" || args[1].startsWith("-")) {
        err.println(USAGE)
        return 2
    }
    return try {
        val facts = RepositoryScanner().scan(Path.of(args[1]))
        out.println("Repository: ${facts.root}")
        facts.builds.forEach { build ->
            out.println("Build: ${build.tool} ${build.version ?: "unknown (no recognized wrapper version)"} [${build.definition}]")
        }
        0
    } catch (e: Exception) {
        err.println("scryer: ${e.message ?: e.javaClass.simpleName}")
        1
    }
}

fun main(args: Array<String>) {
    exitProcess(runCli(args, System.out, System.err))
}
