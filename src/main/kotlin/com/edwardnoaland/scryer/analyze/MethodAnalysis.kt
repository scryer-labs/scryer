package com.edwardnoaland.scryer.analyze

import java.nio.file.Path

enum class MethodChangeKind { ADDED, MODIFIED, DELETED }
data class MethodChange(val kind: MethodChangeKind, val beforePath: String?, val afterPath: String?, val before: JavaMethod?, val after: JavaMethod?)
data class MethodAnalysis(val comparison: GitComparison, val methods: List<MethodChange>, val notes: List<String>)

class MethodAnalyzer {
    fun analyze(comparison: GitComparison): MethodAnalysis = IsolatedSnapshots.use(comparison) { beforeRoot, afterRoot ->
        analyzeSnapshots(comparison, beforeRoot, afterRoot)
    }

    internal fun analyzeSnapshots(comparison: GitComparison, beforeRoot: Path, afterRoot: Path): MethodAnalysis {
        val reader = JavaMethods()
        val changes = mutableListOf<MethodChange>()
        val notes = mutableListOf<String>()
        for (file in comparison.files) {
            val before = readMethods(reader, beforeRoot, file.beforePath).associateBy { it.signature }
            val after = readMethods(reader, afterRoot, file.afterPath).associateBy { it.signature }
            if (file.beforePath?.endsWith(".java") != true && file.afterPath?.endsWith(".java") != true) continue
            val fileChanges = (before.keys + after.keys).mapNotNull { signature ->
                val old = before[signature]
                val new = after[signature]
                val kind = when {
                    old == null -> MethodChangeKind.ADDED
                    new == null -> MethodChangeKind.DELETED
                    old.syntax != new.syntax -> MethodChangeKind.MODIFIED
                    else -> return@mapNotNull null
                }
                MethodChange(kind, file.beforePath, file.afterPath, old, new)
            }
            changes += fileChanges
            notes += "${file.afterPath ?: file.beforePath}: file/class context (imports, fields, initializers, inheritance) is not analyzed for impact yet."
        }
        return MethodAnalysis(comparison, changes, notes)
    }

    private fun readMethods(reader: JavaMethods, root: Path, path: String?): List<JavaMethod> =
        if (path?.endsWith(".java") == true) reader.read(root.resolve(path)) else emptyList()
}
