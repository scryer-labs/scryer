package com.edwardnoaland.scryer.analyze

import com.sun.source.tree.*
import com.sun.source.util.TreePath
import com.sun.source.util.TreeScanner
import com.sun.source.util.Trees
import javax.lang.model.element.Element
import javax.lang.model.element.ExecutableElement
import javax.lang.model.element.Modifier
import javax.lang.model.element.TypeElement
import javax.lang.model.element.VariableElement
import javax.lang.model.type.TypeMirror

/** Bounded straight-line propagation. Unknown control flow and mutable array escapes lose knowledge. */
internal class LiteralReflection(
    private val trees: Trees,
    private val descriptor: (TypeMirror) -> String?,
    private val lookup: (String, String, List<String>, Boolean) -> SymbolId?,
) {
    private sealed interface Value
    private data class ClassValue(val owner: String, val type: String) : Value
    private data class StringValue(val text: String) : Value
    private data class ClassArrayValue(val types: List<String>) : Value
    private data class MethodValue(val id: SymbolId) : Value
    private data class ConstructorValue(val id: SymbolId) : Value

    fun resolve(unit: CompilationUnitTree): Map<MethodInvocationTree, SymbolId> {
        val resolved = mutableMapOf<MethodInvocationTree, SymbolId>()
        fun element(node: Tree, context: CompilationUnitTree = unit): Element? =
            TreePath.getPath(context, node)?.let(trees::getElement)

        fun constant(expression: ExpressionTree, context: CompilationUnitTree = unit): Value? {
            if (expression is ParenthesizedTree) return constant(expression.expression, context)
            if (expression is LiteralTree) return (expression.value as? String)?.let(::StringValue)
            if (expression is MemberSelectTree && expression.identifier.toString() == "class") {
                val path = TreePath.getPath(context, expression.expression) ?: return null
                val type = descriptor(trees.getTypeMirror(path)) ?: return null
                val owner = if (type.startsWith('L')) type.removePrefix("L").removeSuffix(";").replace('/', '.') else expression.expression.toString()
                return ClassValue(owner, type)
            }
            return ((element(expression, context) as? VariableElement)?.constantValue as? String)?.let(::StringValue)
        }

        fun helper(method: ExecutableElement): Value? {
            // No virtual dispatch, parameters, branching, state reads or recursive summaries.
            if (Modifier.STATIC !in method.modifiers || method.parameters.isNotEmpty()) return null
            val path = trees.getPath(method) ?: return null
            val declaration = path.leaf as? MethodTree ?: return null
            val statement = declaration.body?.statements?.singleOrNull() as? ReturnTree ?: return null
            return statement.expression?.let { constant(it, path.compilationUnit) }
        }

        object : TreeScanner<Unit, Unit>() {
            override fun visitMethod(node: MethodTree, unused: Unit?) {
                val values = mutableMapOf<Element, Value>()
                fun evaluate(expression: ExpressionTree): Value? {
                    constant(expression)?.let { return it }
                    if (expression is IdentifierTree) return element(expression)?.let { values[it] }
                    if (expression is ParenthesizedTree) return evaluate(expression.expression)
                    if (expression is NewArrayTree) {
                        val initializers = expression.initializers
                        if (initializers != null) return ClassArrayValue(initializers.map { (evaluate(it) as? ClassValue)?.type ?: return null })
                        if (expression.dimensions.size == 1 && (expression.dimensions.single() as? LiteralTree)?.value == 0) return ClassArrayValue(emptyList())
                        return null
                    }
                    if (expression !is MethodInvocationTree) return null
                    val method = element(expression) as? ExecutableElement ?: return null
                    val owner = (method.enclosingElement as? TypeElement)?.qualifiedName?.toString()
                    val name = method.simpleName.toString()
                    val select = expression.methodSelect as? MemberSelectTree
                    val receiver = select?.expression?.let(::evaluate)
                    val trustedLookup = owner == "java.lang.Class" && name in setOf("getMethod", "getDeclaredMethod", "getConstructor", "getDeclaredConstructor")
                    if (!trustedLookup && expression.arguments.any { evaluate(it) is ClassArrayValue }) {
                        values.entries.removeIf { it.value is ClassArrayValue }
                    }
                    if (owner == "java.lang.Class" && name == "forName" && expression.arguments.size == 1) {
                        val text = (evaluate(expression.arguments.single()) as? StringValue)?.text ?: return null
                        return ClassValue(text, "L${text.replace('.', '/')};")
                    }
                    if (trustedLookup && receiver is ClassValue) {
                        val constructor = name in setOf("getConstructor", "getDeclaredConstructor")
                        val methodName = if (constructor) "<init>" else (expression.arguments.firstOrNull()?.let(::evaluate) as? StringValue)?.text ?: return null
                        val arguments = if (constructor) expression.arguments else expression.arguments.drop(1)
                        val parameters = if (arguments.size == 1 && evaluate(arguments.single()) is ClassArrayValue)
                            (evaluate(arguments.single()) as ClassArrayValue).types
                        else arguments.map { (evaluate(it) as? ClassValue)?.type ?: return null }
                        val id = lookup(receiver.owner, methodName, parameters, name.startsWith("getDeclared")) ?: return null
                        return if (constructor) ConstructorValue(id) else MethodValue(id)
                    }
                    if (owner == "java.lang.reflect.Method" && name == "invoke" && receiver is MethodValue) resolved[expression] = receiver.id
                    if (owner == "java.lang.reflect.Constructor" && name == "newInstance" && receiver is ConstructorValue) resolved[expression] = receiver.id
                    if (owner == "java.lang.Class" && name == "newInstance" && receiver is ClassValue) {
                        lookup(receiver.owner, "<init>", emptyList(), false)?.let { resolved[expression] = it }
                    }
                    return if (expression.arguments.isEmpty()) helper(method) else null
                }
                fun inspect(expression: ExpressionTree) {
                    var unsafe = false
                    object : TreeScanner<Unit, Unit>() {
                        override fun visitAssignment(node: AssignmentTree, unused: Unit?) { unsafe = true }
                        override fun visitCompoundAssignment(node: CompoundAssignmentTree, unused: Unit?) { unsafe = true }
                        override fun visitLambdaExpression(node: LambdaExpressionTree, unused: Unit?) { unsafe = true }
                        override fun visitClass(node: ClassTree, unused: Unit?) { unsafe = true }
                        override fun visitUnary(node: UnaryTree, unused: Unit?) {
                            if (node.kind in setOf(Tree.Kind.PREFIX_INCREMENT, Tree.Kind.POSTFIX_INCREMENT, Tree.Kind.PREFIX_DECREMENT, Tree.Kind.POSTFIX_DECREMENT)) unsafe = true
                            super.visitUnary(node, unused)
                        }
                    }.scan(expression, Unit)
                    if (unsafe) { values.clear(); return }
                    object : TreeScanner<Unit, Unit>() {
                        override fun visitNewClass(node: NewClassTree, unused: Unit?) {
                            if (node.arguments.any { evaluate(it) is ClassArrayValue }) {
                                values.entries.removeIf { it.value is ClassArrayValue }
                            }
                            super.visitNewClass(node, unused)
                        }
                        override fun visitMethodInvocation(node: MethodInvocationTree, unused: Unit?) {
                            evaluate(node)
                            super.visitMethodInvocation(node, unused)
                        }
                    }.scan(expression, Unit)
                }
                fun block(body: BlockTree) {
                    for (statement in body.statements) {
                        when (statement) {
                            is VariableTree -> {
                                statement.initializer?.let(::inspect)
                                val key = element(statement)
                                val value = statement.initializer?.let(::evaluate)
                                if (key != null && value != null) values[key] = value
                            }
                            is ExpressionStatementTree -> inspect(statement.expression)
                            is ReturnTree -> statement.expression?.let(::inspect)
                            is BlockTree -> { block(statement); values.clear() }
                            is TryTree -> {
                                if (statement.resources.isEmpty()) block(statement.block)
                                values.clear()
                            }
                            else -> values.clear()
                        }
                    }
                }
                node.body?.let(::block)
                super.visitMethod(node, unused)
            }
        }.scan(unit, Unit)
        return resolved
    }
}
