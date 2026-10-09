package com.edwardnoaland.scryer.analyze

import com.sun.source.tree.ClassTree
import com.sun.source.tree.MethodTree
import com.sun.source.util.JavacTask
import com.sun.source.util.TreeScanner
import java.nio.file.Path
import javax.tools.Diagnostic
import javax.tools.DiagnosticCollector
import javax.tools.JavaFileObject
import javax.tools.ToolProvider

/** Source-level identities only: types are not resolved against a classpath yet. */
data class JavaMethod(val owner: String, val name: String, val parameterTypes: List<String>, val lines: LineRange, val syntax: String) {
    val signature: String get() = "$owner#$name(${parameterTypes.joinToString(", ")})"
}

internal class JavaMethods {
    fun read(path: Path): List<JavaMethod> {
        require(!java.nio.file.Files.isSymbolicLink(path)) { "Java source is a symlink: $path" }
        val compiler = ToolProvider.getSystemJavaCompiler() ?: error("Method analysis requires a JDK, not a JRE")
        val diagnostics = DiagnosticCollector<JavaFileObject>()
        compiler.getStandardFileManager(diagnostics, null, Charsets.UTF_8).use { manager ->
            val task = compiler.getTask(null, manager, diagnostics, listOf("-proc:none"), null,
                manager.getJavaFileObjects(path.toFile())) as JavacTask
            val units = task.parse().toList()
            val errors = diagnostics.diagnostics.filter { it.kind == Diagnostic.Kind.ERROR }
            check(errors.isEmpty()) { "Cannot parse $path: ${errors.joinToString { "line ${it.lineNumber}: ${it.getMessage(null)}" }}" }
            val positions = com.sun.source.util.Trees.instance(task).sourcePositions
            val result = mutableListOf<JavaMethod>()
            for (unit in units) {
                val owners = mutableListOf<String>()
                val packageName = unit.packageName?.toString().orEmpty()
                object : TreeScanner<Unit, Unit>() {
                    override fun visitClass(node: ClassTree, unused: Unit?) {
                        check(node.simpleName.isNotEmpty()) { "Anonymous class in $path: method identity is not supported yet" }
                        owners += node.simpleName.toString()
                        super.visitClass(node, unused)
                        owners.removeAt(owners.lastIndex)
                    }

                    override fun visitMethod(node: MethodTree, unused: Unit?) {
                        val start = positions.getStartPosition(unit, node)
                        val end = positions.getEndPosition(unit, node)
                        val first = unit.lineMap.getLineNumber(start).toInt()
                        val last = unit.lineMap.getLineNumber(end - 1).toInt()
                        val owner = (listOf(packageName).filter { it.isNotEmpty() } + owners).joinToString(".")
                        result += JavaMethod(owner, node.name.toString(), node.parameters.map { it.type.toString() },
                            LineRange(first, last - first + 1), node.toString())
                        super.visitMethod(node, unused)
                    }
                }.scan(unit, Unit)
            }
            check(result.map { it.signature }.distinct().size == result.size) { "Ambiguous source method identities in $path" }
            return result
        }
    }
}
