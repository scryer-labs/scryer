package com.edwardnoaland.scryer.analyze

import java.nio.file.Path
import java.util.concurrent.TimeUnit

/** A zero-length range represents an insertion/deletion boundary, not an executed line. */
data class LineRange(val start: Int, val count: Int)
data class ChangedLines(val before: LineRange, val after: LineRange)
data class ChangedFile(val status: String, val beforePath: String?, val afterPath: String?, val lines: List<ChangedLines>)
data class GitComparison(val repository: Path, val before: String, val after: String, val files: List<ChangedFile>)

class GitComparer {
    fun compare(directory: Path, beforeRef: String, afterRef: String): GitComparison {
        val root = Path.of(git(directory, "rev-parse", "--show-toplevel").trim())
        val before = resolve(root, beforeRef)
        val after = resolve(root, afterRef)
        val entries = git(root, "diff", "--no-color", "--no-ext-diff", "--no-textconv", "--name-status", "-z", "--find-renames", before, after, "--")
            .split('\u0000').dropLastWhile { it.isEmpty() }
        val files = mutableListOf<ChangedFile>()
        var index = 0
        while (index < entries.size) {
            val status = entries[index++]
            val first = entries[index++]
            val second = if (status.startsWith("R") || status.startsWith("C")) entries[index++] else first
            val oldPath = first.takeUnless { status == "A" }
            val newPath = second.takeUnless { status == "D" }
            val paths = listOfNotNull(oldPath, newPath).distinct()
            val patch = git(root, "diff", "--no-color", "--no-ext-diff", "--no-textconv", "--unified=0", "--find-renames", before, after, "--", *paths.toTypedArray())
            val lines = HUNK.findAll(patch).map { match ->
                ChangedLines(
                    LineRange(match.groupValues[1].toInt(), match.groupValues[2].ifEmpty { "1" }.toInt()),
                    LineRange(match.groupValues[3].toInt(), match.groupValues[4].ifEmpty { "1" }.toInt()),
                )
            }.toList()
            files += ChangedFile(status, oldPath, newPath, lines)
        }
        return GitComparison(root, before, after, files)
    }

    private fun resolve(root: Path, ref: String): String =
        git(root, "rev-parse", "--verify", "--end-of-options", "$ref^{commit}").trim()

    private fun git(directory: Path, vararg args: String): String {
        val output = java.nio.file.Files.createTempFile("scryer-git-", ".log")
        try {
            val process = ProcessBuilder(listOf("git", "--no-pager", "--literal-pathspecs", "-C", directory.toAbsolutePath().toString()) + args)
                .redirectErrorStream(true).redirectOutput(output.toFile()).start()
            if (!process.waitFor(30, TimeUnit.SECONDS)) {
                process.destroyForcibly().waitFor()
                error("Git comparison timed out")
            }
            val text = java.nio.file.Files.readString(output)
            check(process.exitValue() == 0) { "Git command failed: ${text.trim()}" }
            return text
        } finally {
            java.nio.file.Files.deleteIfExists(output)
        }
    }

    private companion object {
        val HUNK = Regex("(?m)^@@ -(\\d+)(?:,(\\d+))? \\+(\\d+)(?:,(\\d+))? @@")
    }
}
