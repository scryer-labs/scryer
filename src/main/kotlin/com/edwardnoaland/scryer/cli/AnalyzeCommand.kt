package com.edwardnoaland.scryer.cli

import java.io.PrintStream
import java.nio.file.Path
import com.edwardnoaland.scryer.analyze.GitComparer
import com.edwardnoaland.scryer.analyze.LineRange

internal const val ANALYZE_USAGE = "Usage: scryer analyze --before <ref> --after <ref>"

private data class AnalyzeOptions(val before: String, val after: String)

internal fun runAnalyzeCommand(args: Array<String>, out: PrintStream, err: PrintStream): Int {
    if (args.contentEquals(arrayOf("--help")) || args.contentEquals(arrayOf("-h"))) {
        out.println(ANALYZE_USAGE)
        return 0
    }

    val options = parseAnalyzeOptions(args)
    if (options == null) {
        err.println(ANALYZE_USAGE)
        return 2
    }

    return try {
        val comparison = GitComparer().compare(Path.of("."), options.before, options.after)
        out.println("Repository: ${comparison.repository}")
        out.println("Before: ${comparison.before}")
        out.println("After: ${comparison.after}")
        out.println("Changed files: ${comparison.files.size}")
        comparison.files.forEach { file ->
            val path = if (file.beforePath != null && file.afterPath != null && file.beforePath != file.afterPath) {
                "${file.beforePath} -> ${file.afterPath}"
            } else file.afterPath ?: file.beforePath
            out.println("${file.status} $path")
            file.lines.forEach { lines ->
                out.println("  before ${formatRange(lines.before)} -> after ${formatRange(lines.after)}")
            }
            if (file.lines.isEmpty()) out.println("  No textual line changes (for example rename, binary or mode change)")
        }
        out.println("Git snapshot differences only; symbols, build and test evidence are not collected yet.")
        0
    } catch (exception: Exception) {
        err.println("scryer: ${exception.message ?: exception.javaClass.simpleName}")
        1
    }
}

private fun formatRange(range: LineRange): String = when (range.count) {
    0 -> "boundary ${range.start} (0 lines)"
    1 -> "line ${range.start}"
    else -> "lines ${range.start}-${range.start + range.count - 1}"
}

/** Parse command arguments before performing Git operations. */
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
