package com.edwardnoaland.scryer.cli

import java.io.PrintStream

internal const val ANALYZE_USAGE = "Usage: scryer analyze --before <ref> --after <ref>"

private data class AnalyzeOptions(val before: String, val after: String)

internal fun runAnalyzeCommand(args: Array<String>, out: PrintStream, err: PrintStream): Int {
    if (args.contentEquals(arrayOf("--help")) || args.contentEquals(arrayOf("-h"))) {
        out.println(ANALYZE_USAGE)
        return 0
    }

    if (parseAnalyzeOptions(args) == null) {
        err.println(ANALYZE_USAGE)
        return 2
    }

    out.println("Analyzing…")
    return 0
}

/** Checks the command shape only. Git reference resolution belongs to the next increment. */
private fun parseAnalyzeOptions(args: Array<String>): AnalyzeOptions? {
    var before: String? = null
    var after: String? = null
    var index = 0

    while (index < args.size) {
        val flag = args[index]
        if (flag !in listOf("--before", "--after")) {
            return null
        }
        if (index + 1 >= args.size) {
            return null
        }
        val reference = args[index + 1]
        if (reference.isBlank() || reference.startsWith('-')) {
            return null
        }

        when (flag) {
            "--before" -> {
                if (before != null) {
                    return null
                }
                before = reference
            }
            "--after" -> {
                if (after != null) {
                    return null
                }
                after = reference
            }
        }
        index += 2
    }

    return AnalyzeOptions(before ?: return null, after ?: return null)
}
