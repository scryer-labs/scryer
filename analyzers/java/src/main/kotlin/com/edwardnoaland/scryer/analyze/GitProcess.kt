package com.edwardnoaland.scryer.analyze

import java.nio.file.Path
import java.util.concurrent.TimeUnit

internal object GitProcess {
    fun run(directory: Path, vararg args: String): String {
        val output = java.nio.file.Files.createTempFile("scryer-git-", ".log")
        try {
            val process = ProcessBuilder(listOf("git", "--no-pager", "--literal-pathspecs", "-C", directory.toAbsolutePath().toString()) + args)
                .redirectErrorStream(true).redirectOutput(output.toFile()).start()
            if (!process.waitFor(30, TimeUnit.SECONDS)) {
                process.destroyForcibly().waitFor()
                error("Git command timed out")
            }
            // Diff bodies can contain non-UTF-8 source bytes; hunk metadata remains ASCII.
            val text = java.nio.file.Files.readAllBytes(output).toString(Charsets.UTF_8)
            check(process.exitValue() == 0) { "Git command failed: ${text.trim()}" }
            return text
        } finally {
            java.nio.file.Files.deleteIfExists(output)
        }
    }
}
