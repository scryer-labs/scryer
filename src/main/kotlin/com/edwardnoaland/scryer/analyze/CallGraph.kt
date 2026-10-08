package com.edwardnoaland.scryer.analyze

import com.edwardnoaland.scryer.analyze.model.AnalysisInputs
import com.edwardnoaland.scryer.analyze.model.AnalysisModule
import com.sun.source.tree.*
import com.sun.source.util.JavacTask
import com.sun.source.util.TreePathScanner
import com.sun.source.util.Trees
import java.nio.file.Files
import java.nio.file.Path
import javax.lang.model.element.ExecutableElement
import javax.lang.model.element.Modifier
import javax.lang.model.element.TypeElement
import javax.lang.model.type.*
import javax.tools.Diagnostic
import javax.tools.DiagnosticCollector
import javax.tools.JavaFileObject
import javax.tools.ToolProvider

/** JVM-style identity, including erased parameters and return type. */
data class SymbolId(val owner: String, val name: String, val descriptor: String) {
    override fun toString(): String = "$owner#$name$descriptor"
}
data class SymbolLocation(val id: SymbolId, val path: String, val line: Int, val sourceSignature: String, val role: SourceRole = sourceRole(path), val module: String? = null)
enum class CallKind { DIRECT, POSSIBLE_DISPATCH, METHOD_REFERENCE, REFLECTION }
data class CallEdge(val caller: SymbolId, val callee: SymbolId, val kind: CallKind)
data class CallBoundary(val caller: SymbolId?, val path: String, val line: Long, val expression: String, val reason: String)
data class CallGraph(val symbols: List<SymbolLocation>, val edges: List<CallEdge>, val boundaries: List<CallBoundary>, val notes: List<String>)

internal class CallGraphCollector {
    fun collect(root: Path, inputs: AnalysisInputs = AnalysisInputs()): CallGraph {
        if (inputs.modules.isEmpty()) return collectSources(root).let { it.copy(notes = inputs.notes + it.notes) }
        val graphs = inputs.modules.map { module ->
            val supporting = linkedSetOf(module.id)
            val pending = ArrayDeque(module.dependencies)
            while (pending.isNotEmpty()) {
                val id = pending.removeFirst()
                if (supporting.add(id)) pending.addAll(inputs.modules.find { it.id == id }?.dependencies.orEmpty())
            }
            val modules = inputs.modules.filter { it.id in supporting }
            val sources = modules.flatMap { it.sources }.map { it.path }
            collectSources(root, sources, module.classpath, modules).let { graph ->
                val owned = graph.symbols.filter { it.module == module.id }.map { it.id }.toSet()
                graph.copy(symbols = graph.symbols.filter { it.module == module.id },
                    edges = graph.edges.filter { it.caller in owned },
                    boundaries = graph.boundaries.filter { boundary ->
                        boundary.caller in owned || (boundary.caller == null && module.sources.any { root.resolve(boundary.path).startsWith(it.path) })
                    }, notes = listOf("Module ${module.id}: ${module.classpath.size} classpath entries; ${module.sources.size} Java source roots.") + module.notes + graph.notes)
            }
        }
        val symbols = graphs.flatMap { it.symbols }.distinct()
        val ambiguous = symbols.groupBy { it.id }.filterValues { it.size > 1 }.keys
        val valid = symbols.filter { it.id !in ambiguous }
        val ids = valid.map { it.id }.toSet()
        val notes = inputs.notes + graphs.flatMap { it.notes } +
            if (ambiguous.isEmpty()) emptyList() else listOf("${ambiguous.size} duplicate source symbol identities across modules excluded rather than merged.")
        return CallGraph(valid, graphs.flatMap { it.edges }.distinct().filter { it.caller in ids && it.callee in ids },
            graphs.flatMap { it.boundaries }.distinct(), notes.distinct())
    }

    private fun collectSources(root: Path, roots: List<Path> = listOf(root), classpath: List<Path> = emptyList(), modules: List<AnalysisModule> = emptyList()): CallGraph {
        val paths = roots.filter { Files.isDirectory(it) && it.normalize().startsWith(root.normalize()) }.flatMap { source -> Files.walk(source).use { stream ->
            stream.filter { Files.isRegularFile(it) && !Files.isSymbolicLink(it) && it.toString().endsWith(".java") }
                .filter { path -> root.relativize(path).none { it.toString() in EXCLUDED } }
                .sorted().toList()
        } }.distinct()
        if (paths.isEmpty()) return CallGraph(emptyList(), emptyList(), emptyList(), emptyList())
        val compiler = ToolProvider.getSystemJavaCompiler() ?: error("Symbol analysis requires a JDK")
        val diagnostics = DiagnosticCollector<JavaFileObject>()
        compiler.getStandardFileManager(diagnostics, null, Charsets.UTF_8).use { manager ->
            val task = compiler.getTask(null, manager, diagnostics, listOf("-proc:none", "-implicit:none", "-classpath", classpath.joinToString(java.io.File.pathSeparator), "-sourcepath", ""), null,
                manager.getJavaFileObjectsFromPaths(paths)) as JavacTask
            val units = task.parse().toList()
            val declaredOwners = mutableListOf<String>()
            for (unit in units) {
                val enclosing = mutableListOf<String>()
                object : com.sun.source.util.TreeScanner<Unit, Unit>() {
                    override fun visitClass(node: ClassTree, unused: Unit?) {
                        enclosing += node.simpleName.toString()
                        val prefix = unit.packageName?.toString()?.let { "$it." }.orEmpty()
                        declaredOwners += prefix + enclosing.joinToString("$")
                        super.visitClass(node, unused)
                        enclosing.removeAt(enclosing.lastIndex)
                    }
                }.scan(unit, Unit)
            }
            val duplicateOwners = declaredOwners.groupingBy { it }.eachCount().filterValues { it > 1 }.keys
            task.analyze().toList()
            val trees = Trees.instance(task)
            val types = task.types
            val elements = task.elements
            fun descriptor(type: TypeMirror): String? = when (type.kind) {
                TypeKind.VOID -> "V"
                TypeKind.BOOLEAN -> "Z"
                TypeKind.BYTE -> "B"
                TypeKind.SHORT -> "S"
                TypeKind.INT -> "I"
                TypeKind.LONG -> "J"
                TypeKind.CHAR -> "C"
                TypeKind.FLOAT -> "F"
                TypeKind.DOUBLE -> "D"
                TypeKind.ARRAY -> descriptor((type as ArrayType).componentType)?.let { "[$it" }
                TypeKind.DECLARED -> "L${elements.getBinaryName((type as DeclaredType).asElement() as TypeElement).toString().replace('.', '/')};"
                TypeKind.TYPEVAR -> descriptor(types.erasure(type))
                else -> null
            }
            fun symbol(method: ExecutableElement): SymbolId? {
                val owner = method.enclosingElement as? TypeElement ?: return null
                val parameters = method.parameters.map { descriptor(types.erasure(it.asType())) ?: return null }
                val result = descriptor(types.erasure(method.returnType)) ?: return null
                return SymbolId(elements.getBinaryName(owner).toString(), method.simpleName.toString(), "(${parameters.joinToString("")})$result")
            }
            val methods = linkedMapOf<SymbolId, ExecutableElement>()
            val locations = mutableListOf<SymbolLocation>()
            val boundaries = mutableListOf<CallBoundary>()
            val edges = linkedSetOf<CallEdge>()
            for (unit in units) {
                val path = root.relativize(Path.of(unit.sourceFile.toUri())).toString()
                object : TreePathScanner<Unit, Unit>() {
                    override fun visitMethod(node: MethodTree, unused: Unit?) {
                        val position = trees.sourcePositions.getStartPosition(unit, node)
                        if (position >= 0) {
                            val method = trees.getElement(currentPath) as? ExecutableElement
                            val id = method?.let(::symbol)
                            if (id != null) {
                                methods[id] = method
                                val owner = method.enclosingElement as TypeElement
                                val sourceSignature = "${owner.qualifiedName}#${node.name}(${node.parameters.joinToString(", ") { it.type.toString() }})"
                                val sourcePath = root.resolve(path)
                                val module = modules.firstOrNull { candidate -> candidate.sources.any { sourcePath.startsWith(it.path) } }
                                val role = module?.sources?.filter { sourcePath.startsWith(it.path) }?.maxByOrNull { it.path.nameCount }?.role ?: sourceRole(path)
                                locations += SymbolLocation(id, path, unit.lineMap.getLineNumber(position).toInt(), sourceSignature, role, module?.id)
                            } else boundaries += CallBoundary(null, path, unit.lineMap.getLineNumber(position), node.name.toString(), "Unresolved declaration signature")
                        }
                        super.visitMethod(node, unused)
                    }
                }.scan(unit, Unit)
            }
            val ambiguous = locations.groupBy { it.id }.filterValues { it.size > 1 }.keys +
                methods.keys.filter { it.owner in duplicateOwners }
            ambiguous.forEach { methods.remove(it) }
            val reflection = LiteralReflection(trees, ::descriptor) { ownerName, name, parameters, declared ->
                val owner = elements.getTypeElement(ownerName.replace('$', '.'))
                val candidates = if (owner == null) emptyList() else
                    (if (declared) owner.enclosedElements else elements.getAllMembers(owner))
                        .filterIsInstance<ExecutableElement>()
                        .filter { it.simpleName.toString() == name && (declared || Modifier.PUBLIC in it.modifiers) }
                        .mapNotNull(::symbol)
                        .filter { it in methods && it.descriptor.startsWith("(${parameters.joinToString("")})") }
                candidates.distinct().singleOrNull()
            }
            for (unit in units) {
                val reflected = reflection.resolve(unit)
                val path = root.relativize(Path.of(unit.sourceFile.toUri())).toString()
                var caller: SymbolId? = null
                object : TreePathScanner<Unit, Unit>() {
                    override fun visitClass(node: ClassTree, unused: Unit?) {
                        val previous = caller
                        caller = null
                        super.visitClass(node, unused)
                        caller = previous
                    }
                    override fun visitMethod(node: MethodTree, unused: Unit?) {
                        val previous = caller
                        val position = trees.sourcePositions.getStartPosition(unit, node)
                        caller = if (position >= 0) (trees.getElement(currentPath) as? ExecutableElement)?.let(::symbol)?.takeIf { it in methods } else null
                        super.visitMethod(node, unused)
                        caller = previous
                    }
                    private fun record(node: Tree, reference: Boolean = false, superCall: Boolean = false) {
                        val position = trees.sourcePositions.getStartPosition(unit, node)
                        if (position < 0) return
                        val reflectedId = (node as? MethodInvocationTree)?.let { reflected[it] }
                        val target = reflectedId?.let { methods[it] } ?: (trees.getElement(currentPath) as? ExecutableElement)
                        val id = reflectedId ?: target?.let(::symbol)
                        val source = caller
                        if (source == null || id == null || id !in methods) {
                            boundaries += CallBoundary(source, path, unit.lineMap.getLineNumber(position), node.toString(),
                                if (source == null) "Initializer or unresolved caller" else if (id == null) "Unresolved call" else if (id in ambiguous) "Ambiguous source target: $id" else "External target: $id")
                            return
                        }
                        edges += CallEdge(source, id, if (reflectedId != null) CallKind.REFLECTION else if (reference) CallKind.METHOD_REFERENCE else CallKind.DIRECT)
                        if (!superCall && target != null && target.simpleName.toString() != "<init>" &&
                            target.modifiers.none { it in setOf(Modifier.STATIC, Modifier.PRIVATE, Modifier.FINAL) }) {
                            for ((candidateId, candidate) in methods) {
                                val owner = candidate.enclosingElement as? TypeElement ?: continue
                                if (candidateId != id && elements.overrides(candidate, target, owner)) {
                                    edges += CallEdge(source, candidateId, CallKind.POSSIBLE_DISPATCH)
                                }
                            }
                        }
                    }
                    override fun visitMethodInvocation(node: MethodInvocationTree, unused: Unit?) {
                        val select = node.methodSelect as? MemberSelectTree
                        record(node, superCall = select?.expression?.toString()?.let { it == "super" || it.endsWith(".super") } == true)
                        super.visitMethodInvocation(node, unused)
                    }
                    override fun visitNewClass(node: NewClassTree, unused: Unit?) {
                        record(node)
                        super.visitNewClass(node, unused)
                    }
                    override fun visitMemberReference(node: MemberReferenceTree, unused: Unit?) {
                        record(node, reference = true, superCall = node.qualifierExpression.toString().let { it == "super" || it.endsWith(".super") })
                        super.visitMemberReference(node, unused)
                    }
                }.scan(unit, Unit)
            }
            val errors = diagnostics.diagnostics.filter { it.kind == Diagnostic.Kind.ERROR }
            val notes = mutableListOf("Partial source call graph: build inputs are best effort; no annotation processing, generated sources, target JDK boot API equivalence, framework wiring or dynamic reflection resolution. Literal Class.forName/getMethod/invoke chains are supported.",
                "Virtual dispatch includes possible source overrides; lambda/method-reference edges are potential calls, not execution evidence.")
            if (errors.isNotEmpty()) notes += "Javac attribution reported ${errors.size} errors; unresolved calls/signatures remain boundaries. Example: ${errors.first().getMessage(null)}"
            if (ambiguous.isNotEmpty()) notes += "${ambiguous.size} duplicate source symbol identities were excluded; ambiguous identities are not merged."
            return CallGraph(locations.filter { it.id !in ambiguous }.distinct(), edges.toList(), boundaries, notes)
        }
    }

    private companion object {
        val EXCLUDED = setOf(".git", ".tooling", ".gradle", "build", "target", "node_modules")
    }
}
