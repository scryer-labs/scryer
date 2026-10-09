package com.edwardnoaland.scryer.analyze

import java.nio.file.Files
import java.nio.file.Path
import java.util.Comparator

/** Worktree metadata belongs to a temporary clone, never the user's repository. */
internal object IsolatedSnapshots {
    fun <T> use(comparison: GitComparison, action: (Path, Path) -> T): T {
        val temporary = Files.createTempDirectory("scryer-snapshots-")
        try {
            val clone = temporary.resolve("repository")
            GitProcess.run(temporary, "clone", "--shared", "--no-checkout", "--", comparison.repository.toString(), clone.toString())
            val before = temporary.resolve("before")
            val after = temporary.resolve("after")
            for ((ref, path) in listOf(comparison.before to before, comparison.after to after)) {
                GitProcess.run(clone, "-c", "core.hooksPath=/dev/null", "worktree", "add", "--detach", path.toString(), ref)
            }
            return action(before, after)
        } finally {
            Files.walk(temporary).use { paths ->
                paths.sorted(Comparator.reverseOrder()).forEach(Files::delete)
            }
        }
    }
}
